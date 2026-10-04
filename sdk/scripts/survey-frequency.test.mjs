// Survey display rules against the built bundle. Run: npm run build && npm test
import assert from 'node:assert';
const store = new Map();
globalThis.localStorage = { getItem: k => store.get(k) ?? null, setItem: (k, v) => store.set(k, String(v)) };
globalThis.document = { getElementById: () => null };
const { AnalyticsCore } = await import(new URL('../dist/analytics.esm.js', import.meta.url));
const P = AnalyticsCore.prototype;
let shown = 0;
const ctx = { STORAGE_PREFIX: 'sj_', userId: 'u1', showSurvey: () => shown++, inSurveySample: P.inSurveySample, getUserId: () => 'u1' };
const show = s => { const before = shown; P.showSurveyOnce.call(ctx, s); return shown > before; };
const day = 86400000, realNow = Date.now;
const at = d => (Date.now = () => realNow() + d * day);

at(0); const once = { id: 1, frequency: 'once' };
assert(show(once)); at(400); assert(!show(once), 'once never repeats');

at(0); const until = { id: 2, frequency: 'until_answered', repeat_days: 7 };
assert(show(until)); at(3); assert(!show(until), 'waits repeat_days'); at(8); assert(show(until), 'shows again unanswered');
store.set('sj_survey_2_answered', '1'); at(30); assert(!show(until), 'stops once answered');

at(0); const rec = { id: 3, frequency: 'recurring', repeat_days: 30 };
store.set('sj_survey_3_answered', '1');
assert(show(rec)); at(10); assert(!show(rec)); at(31); assert(show(rec), 'recurring ignores answered');

at(0); store.set('sj_survey_4', '1'); assert(!show({ id: 4 }), "legacy '1' + default once stays hidden");

let inSample = 0;
for (let i = 0; i < 2000; i++) inSample += P.inSurveySample.call({ userId: 'user' + i }, { id: 9, sample_percent: 25 }) ? 1 : 0;
assert(inSample > 400 && inSample < 600, 'sample ~25%: ' + inSample);
assert.equal(P.inSurveySample.call({ userId: 'x' }, { id: 9, sample_percent: 25 }), P.inSurveySample.call({ userId: 'x' }, { id: 9, sample_percent: 25 }));
console.log('ok', inSample);
