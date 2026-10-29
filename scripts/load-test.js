import http from 'k6/http';
import exec from 'k6/execution';
import { check, fail } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

function positiveInteger(name, fallback, maximum) {
  const value = Number(__ENV[name] || fallback);
  if (!Number.isInteger(value) || value < 1 || value > maximum) {
    throw new Error(`${name} must be an integer between 1 and ${maximum}`);
  }
  return value;
}

const base = __ENV.BASE_URL || 'http://app:8080/api/v1';
const rate = positiveInteger('LOAD_RATE', 100, 2000);
const seconds = positiveInteger('LOAD_SECONDS', 60, 600);
const conflictSeconds = positiveInteger('LOAD_CONFLICT_SECONDS', 20, 120);
if ((rate * seconds) % 10 !== 0) throw new Error('LOAD_RATE * LOAD_SECONDS must be divisible by 10');
const listCount = 20;
const raceCases = 20;
const expectedReservations = rate * seconds / 10;
const eventDate = '2030-06-15';
const warmupSeconds = 5;
const drainSeconds = 11; // Longer than the 10s HTTP timeout; phases do not overlap.
const mixedRequests = new Counter('mixed_requests');
const mixedOK = new Rate('mixed_ok');
const requestMS = new Trend('request_ms', true);
const conflictOK = new Rate('conflict_ok');
const conflictRequests = new Counter('conflict_requests');
const raceOK = new Rate('race_ok');
const raceCount = new Counter('race_cases');
const expected2xx = http.expectedStatuses(200, 201);
const expected409 = http.expectedStatuses(409);

function arrival(duration, startTime, fn) {
  return {
    executor: 'constant-arrival-rate', rate, timeUnit: '1s',
    duration: `${duration}s`, startTime: `${startTime}s`, exec: fn,
    preAllocatedVUs: 50, maxVUs: 200, gracefulStop: '11s',
  };
}

const thresholds = {
  dropped_iterations: ['count==0'],
  // k6 may start one extra iteration at the boundary; under-delivery is a failure.
  mixed_requests: [`count>=${rate * seconds}`, `count<=${rate * seconds + 1}`],
  mixed_ok: ['rate==1'],
  request_ms: ['p(95)<200'],
  conflict_ok: ['rate==1'],
  conflict_requests: [`count>=${rate * conflictSeconds}`, `count<=${rate * conflictSeconds + 1}`],
  race_ok: ['rate==1'],
  race_cases: [`count==${raceCases}`],
  'checks{phase:fixture}': ['rate==1'],
  'checks{phase:integrity}': ['rate==1'],
};
for (const operation of ['public_read', 'owner_list', 'wishlist_patch', 'item_patch', 'reserve']) {
  thresholds[`mixed_ok{operation:${operation}}`] = ['rate==1'];
  thresholds[`request_ms{operation:${operation}}`] = ['p(95)<200'];
}

export const options = {
  setupTimeout: '120s', teardownTimeout: '60s', batch: 20, batchPerHost: 20,
  systemTags: ['status', 'method', 'name', 'scenario', 'expected_response'],
  summaryTrendStats: ['avg', 'med', 'p(95)', 'p(99)', 'max'],
  scenarios: {
    warmup: arrival(warmupSeconds, 0, 'warmup'),
    mixed: arrival(seconds, warmupSeconds + drainSeconds, 'mixed'),
    conflicts: arrival(conflictSeconds, warmupSeconds + seconds + 2 * drainSeconds, 'conflicts'),
    races: {
      executor: 'shared-iterations', vus: 1, iterations: raceCases,
      startTime: `${warmupSeconds + seconds + conflictSeconds + 3 * drainSeconds}s`,
      maxDuration: '60s', exec: 'race',
    },
  },
  thresholds,
};

function params(token, name, callback = expected2xx) {
  const headers = { 'Content-Type': 'application/json' };
  if (token) headers.Authorization = `Bearer ${token}`;
  return { headers, tags: { name }, timeout: '10s', redirects: 0, responseCallback: callback };
}

function request(method, path, token, body, name, callback) {
  return http.request(method, base + path, body === null ? null : JSON.stringify(body), params(token, name, callback));
}

function json(response) {
  try {
    const value = response.json();
    return value && typeof value === 'object' ? value : {};
  } catch (_) { return {}; }
}

function requireThat(condition, message, phase = 'fixture') {
  if (!check(condition, { [message]: (ok) => Boolean(ok) }, { phase })) fail(message);
}

function fixture(method, path, token, body, expected) {
  const response = request(method, path, token, body, 'fixture');
  requireThat(response.status === expected, `${method} fixture status ${response.status}, expected ${expected}`);
  return json(response);
}

function listPath(list) { return `/wishlists/${list.id}`; }
function sharedPath(list) { return `/shared/${list.share_token}`; }
function reservePath(list, itemID) { return `${sharedPath(list)}/items/${itemID}/reserve`; }

function validItems(body, list) {
  return Array.isArray(body.items) && body.items.length === list.itemIDs.length
    && new Set(body.items.map((item) => item?.id)).size === list.itemIDs.length
    && body.items.every((item) => item && list.itemIDs.includes(item.id)
      && item.wishlist_id === list.id && typeof item.is_reserved === 'boolean');
}

export function setup() {
  const token = fixture('POST', '/auth/register', '', { email: 'load@example.com', password: 'load-password-123' }, 201).token;
  requireThat(typeof token === 'string' && token.length > 0, 'registration token exists');
  const lists = [], races = [], reservationItems = [];
  for (let i = 0; i < listCount + raceCases; i++) {
    const list = fixture('POST', '/wishlists', token, { title: `list-${i}`, description: 'original', event_date: eventDate }, 201);
    requireThat(list.id && list.share_token, 'wishlist has ID and share token');
    const item = fixture('POST', `${listPath(list)}/items`, token,
      { title: 'original gift', description: 'original', url: 'https://example.com/gift', priority: 3 }, 201);
    requireThat(item.id && item.wishlist_id === list.id && item.is_reserved === false, 'gift starts unreserved');
    const entry = { ...list, protectedID: item.id, itemIDs: [item.id] };
    if (i >= listCount) {
      races.push(entry);
    } else {
      const reserved = fixture('POST', reservePath(list, item.id), '', null, 200);
      requireThat(reserved.id === item.id && reserved.is_reserved === true, 'protected gift is reserved');
      lists.push(entry);
    }
  }
  for (let i = 0; i < expectedReservations; i++) {
    const list = lists[i % listCount];
    const item = fixture('POST', `${listPath(list)}/items`, token, { title: `gift-${i}` }, 201);
    requireThat(item.id && item.wishlist_id === list.id && item.is_reserved === false, 'workload gift starts unreserved');
    list.itemIDs.push(item.id);
    reservationItems.push(item.id);
  }
  return { token, lists, races, reservationItems };
}

export function warmup(data) {
  const list = data.lists[exec.scenario.iterationInTest % listCount];
  const response = request('GET', sharedPath(list), '', null, 'warmup');
  requireThat(response.status === 200 && validItems(json(response), list), 'warmup succeeds');
}

export function mixed(data) {
  const index = exec.scenario.iterationInTest, cycle = Math.floor(index / 10), kind = index % 10;
  const list = data.lists[cycle % listCount];
  let method = 'GET', path, body = null, token = data.token, operation, valid;
  if (kind < 5) {
    path = sharedPath(list); token = ''; operation = 'public_read';
    valid = (r, b) => r.status === 200 && b.id === list.id && b.event_date === eventDate
      && !('user_id' in b) && !('share_token' in b) && validItems(b, list);
  } else if (kind < 7) {
    path = '/wishlists'; operation = 'owner_list';
    const ids = [...data.lists, ...data.races].map((entry) => entry.id);
    valid = (r, b) => r.status === 200 && Array.isArray(b) && b.length === ids.length
      && new Set(b.map((entry) => entry?.id)).size === ids.length
      && b.every((entry) => entry && ids.includes(entry.id) && entry.event_date === eventDate);
  } else if (kind === 7) {
    path = listPath(list); method = 'PATCH'; operation = 'wishlist_patch'; body = { description: 'edited' };
    valid = (r, b) => r.status === 200 && b.id === list.id && b.description === 'edited' && b.title === list.title && b.event_date === eventDate;
  } else if (kind === 8) {
    path = `${listPath(list)}/items/${list.protectedID}`; method = 'PATCH'; operation = 'item_patch'; body = { description: 'edited' };
    valid = (r, b) => r.status === 200 && b.id === list.protectedID && b.description === 'edited'
      && b.title === 'original gift' && b.is_reserved === true;
  } else {
    path = reservePath(list, data.reservationItems[cycle]); method = 'POST'; token = ''; operation = 'reserve';
    valid = (r, b) => r.status === 200 && b.id === data.reservationItems[cycle] && b.wishlist_id === list.id && b.is_reserved === true;
  }
  const start = Date.now();
  const response = request(method, path, token, body, operation);
  requestMS.add(Date.now() - start, { operation });
  mixedRequests.add(1);
  let ok = false;
  try { ok = Boolean(valid(response, json(response))); } catch (_) { /* Contract violation is a failed sample. */ }
  mixedOK.add(ok, { operation });
}

export function conflicts(data) {
  const list = data.lists[0];
  const response = request('POST', reservePath(list, list.protectedID), '', null, 'reserve_conflict', expected409);
  conflictRequests.add(1);
  conflictOK.add(response.status === 409 && json(response).error === 'item already reserved');
}

export function race(data) {
  const index = exec.scenario.iterationInTest, list = data.races[index];
  const changes = [
    [listPath(list), list.id, 'title', `race-title-${index}`],
    [listPath(list), list.id, 'description', `race-description-${index}`],
    [`${listPath(list)}/items/${list.protectedID}`, list.protectedID, 'title', `race-gift-${index}`],
    [`${listPath(list)}/items/${list.protectedID}`, list.protectedID, 'description', `race-gift-description-${index}`],
  ];
  const requests = Array.from({ length: 8 }, () => ({
    method: 'POST', url: base + reservePath(list, list.protectedID),
    params: params('', 'race_reserve', http.expectedStatuses(200, 409)),
  }));
  for (const [path, , field, value] of changes) {
    requests.push({ method: 'PATCH', url: base + path, body: JSON.stringify({ [field]: value }), params: params(data.token, 'race_patch') });
  }
  const responses = http.batch(requests);
  let valid = false;
  try {
    const bookings = responses.slice(0, 8);
    valid = responses.length === 12 && bookings.filter((r) => r.status === 200).length === 1
      && bookings.every((r) => r.status === 200
        ? json(r).id === list.protectedID && json(r).is_reserved === true
        : r.status === 409 && json(r).error === 'item already reserved')
      && responses.slice(8).every((r, i) => r.status === 200 && json(r).id === changes[i][1] && json(r)[changes[i][2]] === changes[i][3]);
  } catch (_) { /* Malformed responses must not skip the failure metric. */ }
  raceCount.add(1);
  raceOK.add(Boolean(valid));
}

export function teardown(data) {
  for (let i = 0; i < data.lists.length; i++) {
    const list = data.lists[i];
    const response = request('GET', listPath(list), data.token, null, 'integrity');
    const current = json(response);
    const description = i < expectedReservations ? 'edited' : 'original';
    requireThat(response.status === 200 && current.id === list.id && validItems(current, list), 'workload gifts remain readable', 'integrity');
    requireThat(current.title === list.title && current.description === description && current.event_date === eventDate,
      'wishlist patch persists without changing other fields', 'integrity');
    requireThat(current.items.every((item) => item.is_reserved === true), 'all planned and original reservations persist', 'integrity');
    const protectedItem = current.items.find((item) => item.id === list.protectedID);
    requireThat(protectedItem.title === 'original gift' && protectedItem.description === description
      && protectedItem.url === 'https://example.com/gift' && protectedItem.priority === 3,
    'gift patch persists without losing original fields or reservation', 'integrity');
  }
  for (let i = 0; i < data.races.length; i++) {
    const list = data.races[i];
    const response = request('GET', listPath(list), data.token, null, 'integrity');
    const current = json(response);
    requireThat(response.status === 200 && current.id === list.id && validItems(current, list), 'race gift remains readable', 'integrity');
    requireThat(current.title === `race-title-${i}` && current.description === `race-description-${i}` && current.event_date === eventDate,
      'concurrent wishlist patches both persist', 'integrity');
    const item = current.items[0];
    requireThat(item.title === `race-gift-${i}` && item.description === `race-gift-description-${i}` && item.is_reserved === true
      && item.url === 'https://example.com/gift' && item.priority === 3,
    'concurrent item patches and reservation all persist', 'integrity');
  }
}

export function handleSummary(data) {
  const result = {
    measuredAt: new Date().toISOString(), config: { rate, seconds, conflictSeconds, listCount, raceCases, expectedReservations },
    state: data.state, metrics: data.metrics, options: data.options,
  };
  const lines = Object.entries(data.metrics)
    .filter(([name]) => /^(mixed_|conflict_|race_|request_ms|dropped_iterations|checks\{)/.test(name))
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([name, metric]) => `${name}: ${JSON.stringify(metric.values)}`);
  return { '/tmp/summary.json': JSON.stringify(result, null, 2), stdout: '\n' + lines.join('\n') + '\n' };
}
