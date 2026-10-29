import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';
import { createContext, SourceTextModule, SyntheticModule } from 'node:vm';

// Exercise the real scenario. Bad responses must emit failures rather than
// throw before recording a sample and accidentally produce a green result.
async function harness(response, iteration = 0, env = {}) {
  const samples = new Map(), requests = [];
  class Metric {
    constructor(name) { this.name = name; }
    add(value) {
      const values = samples.get(this.name) || [];
      values.push(value);
      samples.set(this.name, values);
    }
  }
  const http = {
    expectedStatuses: () => ({}),
    request: (...args) => { requests.push(args); return typeof response === 'function' ? response(...args) : response; },
    batch: (args) => { requests.push(...args); return response; },
  };
  const context = createContext({ __ENV: env });
  const dependencies = {
    'k6/http': { default: http },
    'k6/execution': { default: { scenario: { iterationInTest: iteration } } },
    k6: { check: (value) => Boolean(value), fail: (message) => { throw new Error(message); } },
    'k6/metrics': { Counter: Metric, Rate: Metric, Trend: Metric },
  };
  const module = new SourceTextModule(await readFile(new URL('./load-test.js', import.meta.url), 'utf8'), { context });
  await module.link((name) => {
    const exports = dependencies[name];
    assert.ok(exports, `unexpected dependency ${name}`);
    return new SyntheticModule(Object.keys(exports), function () {
      for (const [key, value] of Object.entries(exports)) this.setExport(key, value);
    }, { context });
  });
  await module.evaluate();
  const entry = (i) => ({ id: `list-${i}`, title: `list-${i}`, share_token: `share-${i}`, protectedID: `item-${i}`, itemIDs: [`item-${i}`] });
  const data = {
    token: 'sensitive-token', lists: Array.from({ length: 20 }, (_, i) => entry(i)),
    races: Array.from({ length: 20 }, (_, i) => entry(i + 20)), reservationItems: ['new-item'],
  };
  return { api: module.namespace, samples, requests, data };
}

const response = (status, body) => ({ status, json: () => body });
for (const [name, iteration, status, body] of [
  ['null public list', 0, 200, null],
  ['null gift', 0, 200, { id: 'list-0', event_date: '2030-06-15', items: [null] }],
  ['wrong owner list type', 5, 200, {}],
  ['missing wishlist patch', 7, 200, null],
  ['missing item patch', 8, 200, null],
  ['missing reservation', 9, 200, null],
  ['server failure', 0, 503, { error: 'unavailable' }],
  ['unexpected conflict', 9, 409, { error: 'item already reserved' }],
]) {
  test(`${name} records a failed measured request`, async () => {
    const h = await harness(response(status, body), iteration);
    h.api.mixed(h.data);
    assert.equal(h.requests.length, 1);
    assert.deepEqual(h.samples.get('mixed_requests'), [1]);
    assert.deepEqual(h.samples.get('mixed_ok'), [false]);
  });
}

test('invalid JSON records a failed request', async () => {
  const h = await harness({ status: 200, json() { throw new Error('invalid JSON'); } });
  h.api.mixed(h.data);
  assert.deepEqual(h.samples.get('mixed_ok'), [false]);
});

test('valid public read is counted once and carries no JWT', async () => {
  const h = await harness(response(200, { id: 'list-0', event_date: '2030-06-15', items: [{ id: 'item-0', wishlist_id: 'list-0', is_reserved: true }] }));
  h.api.mixed(h.data);
  assert.deepEqual(h.samples.get('mixed_ok'), [true]);
  assert.equal(h.requests.length, 1);
  assert.equal(h.requests[0][3].headers.Authorization, undefined);
});

for (const body of [null, {}, { error: 'other error' }]) {
  test(`wrong 409 body fails: ${JSON.stringify(body)}`, async () => {
    const h = await harness(response(409, body));
    h.api.conflicts(h.data);
    assert.deepEqual(h.samples.get('conflict_requests'), [1]);
    assert.deepEqual(h.samples.get('conflict_ok'), [false]);
  });
}

function raceResponses(winners = 1) {
  const reserved = response(200, { id: 'item-20', is_reserved: true });
  const conflict = response(409, { error: 'item already reserved' });
  return [
    ...Array(winners).fill(reserved), ...Array(8 - winners).fill(conflict),
    response(200, { id: 'list-20', title: 'race-title-0' }),
    response(200, { id: 'list-20', description: 'race-description-0' }),
    response(200, { id: 'item-20', title: 'race-gift-0' }),
    response(200, { id: 'item-20', description: 'race-gift-description-0' }),
  ];
}

for (const winners of [0, 1, 2]) {
  test(`race requires exactly one winner: ${winners}`, async () => {
    const h = await harness(raceResponses(winners));
    h.api.race(h.data);
    assert.deepEqual(h.samples.get('race_ok'), [winners === 1]);
    assert.deepEqual(h.samples.get('race_cases'), [1]);
    assert.equal(h.requests.length, 12);
  });
}

test('successful HTTP status with wrong PATCH value fails a race', async () => {
  const replies = raceResponses();
  replies[9] = response(200, { id: 'list-20', description: 'original' });
  const h = await harness(replies);
  h.api.race(h.data);
  assert.deepEqual(h.samples.get('race_ok'), [false]);
});

test('teardown rejects a lost concurrent PATCH after successful write responses', async () => {
  const h = await harness(response(200, {
    id: 'list-20', title: 'race-title-0', description: 'original', event_date: '2030-06-15',
    items: [{ id: 'item-20', wishlist_id: 'list-20', is_reserved: true }],
  }));
  h.data.lists = [];
  h.data.races = h.data.races.slice(0, 1);
  assert.throws(() => h.api.teardown(h.data), /concurrent wishlist patches both persist/);
});

test('teardown rejects a lost reservation', async () => {
  const h = await harness(response(200, {
    id: 'list-0', title: 'list-0', description: 'edited', event_date: '2030-06-15',
    items: [{ id: 'item-0', wishlist_id: 'list-0', is_reserved: false }],
  }));
  h.data.lists = h.data.lists.slice(0, 1);
  h.data.races = [];
  assert.throws(() => h.api.teardown(h.data), /all planned and original reservations persist/);
});

test('teardown accepts persisted patches, reservations and unchanged fields', async () => {
  const h = await harness((_method, url) => {
    const race = url.endsWith('list-20');
    return response(200, {
      id: race ? 'list-20' : 'list-0', title: race ? 'race-title-0' : 'list-0',
      description: race ? 'race-description-0' : 'edited', event_date: '2030-06-15',
      items: [{ id: race ? 'item-20' : 'item-0', wishlist_id: race ? 'list-20' : 'list-0', is_reserved: true,
        title: race ? 'race-gift-0' : 'original gift', description: race ? 'race-gift-description-0' : 'edited',
        url: 'https://example.com/gift', priority: 3 }],
    });
  });
  h.data.lists = h.data.lists.slice(0, 1);
  h.data.races = h.data.races.slice(0, 1);
  assert.doesNotThrow(() => h.api.teardown(h.data));
  assert.equal(h.requests.length, 2);
});

for (const env of [{ LOAD_RATE: '0' }, { LOAD_SECONDS: 'NaN' }, { LOAD_RATE: '3', LOAD_SECONDS: '1' }]) {
  test(`invalid configuration is rejected: ${JSON.stringify(env)}`, async () => {
    await assert.rejects(() => harness({}, 0, env));
  });
}

test('summary omits fixture credentials', async () => {
  const h = await harness({});
  const output = h.api.handleSummary({ metrics: {}, setup_data: h.data });
  assert.ok(!output['/tmp/summary.json'].includes('sensitive-token'));
  assert.ok(!output['/tmp/summary.json'].includes('share-0'));
});
