'use strict';
const assert = require('node:assert/strict');
const { draft, redact, issueLink } = require('../../ui/static/feedback.js');
const source = {category:'bug', title:'Broken export', steps:'Click Export', actual:'No download', expected:'Download'};
const d = draft(source);
assert.match(d.body, /Reproduction steps/);
assert.match(issueLink('https://git.example.org/team/rd',d.title,d.body),/issues\/new\?/);
assert.throws(()=>draft({title:'',steps:'hello'}),/title/);
assert.throws(()=>draft({title:'hey',steps:'click',logs:'abc',logConsent:false}),/consent/);
assert.match(redact('Authorization: Bearer abcdefg123456'),/\[REDACTED\]/);
assert.match(redact('password=abcdefghijk'),/\[REDACTED\]/);
assert.doesNotMatch(draft({...source,title:'password=abcdefghijk'}).title,/abcdefghijk/);
assert.ok(!issueLink('https://git.example.org/team/rd','t','a'.repeat(2000)).includes('body='));
console.log('feedback client contract: PASS');

// A scan may link to the local composer using only a non-secret opaque ID.
// Test its browser prefill without loading a real forge or contacting a server.
const fs = require('node:fs');
const vm = require('node:vm');
for (const [query, expected] of [
  ['?scan_id=scan-123456789', 'scan-123456789'],
  ['?scan_id=%3Cscript%3E', ''],
  ['?scan_id=' + 'X'.repeat(130), ''],
]) {
  const scan = {value: ''};
  const noop = {addEventListener() {}};
  const form = {dataset: {forgeUrl: 'https://git.example.org/team/rd'}};
  const document = {
    readyState: 'complete',
    getElementById(id) {
      if (id === 'rd-feedback-form') return form;
      if (id === 'rd-fb-scan') return scan;
      if (id === 'rd-fb-open') return null;
      return noop;
    },
  };
  vm.runInNewContext(fs.readFileSync(require.resolve('../../ui/static/feedback.js'), 'utf8'),
      {window: {document, location: {search: query}}, URLSearchParams, URL});
  assert.equal(scan.value, expected);
}
console.log('feedback contextual scan ID: PASS');
