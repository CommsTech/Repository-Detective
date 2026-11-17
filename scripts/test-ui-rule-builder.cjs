#!/usr/bin/env node
// Lightweight dependency-free interactive regression for the settings rule
// builder. This is a DOM contract test, NOT a browser accessibility test.
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const assert = require('node:assert/strict');

function el(initial = '') {
  const listeners = {};
  return {
    value: initial, textContent: '', disabled: false, children: [],
    addEventListener(name, cb) { listeners[name] = cb; },
    click() { listeners.click?.(); },
    setAttribute() {},
    append(...nodes) { this.children.push(...nodes); },
    appendChild(node) { this.children.push(node); },
    remove() {},
    replaceChildren() { this.children.length = 0; }
  };
}
const ids = { id: el(), title: el(), pattern: el(), path: el(), category: el('quality'), severity: el('low') };
const fields = {
  '[data-rule-id]': ids.id,
  '[data-rule-title]': ids.title,
  '[data-rule-pattern]': ids.pattern,
  '[data-rule-path]': ids.path,
  '[data-rule-category]': ids.category,
  '[data-rule-severity]': ids.severity,
  '[data-rule-status]': el(),
  '[data-rule-json]': el(),
  '[data-rule-list]': el(),
  '[data-rule-download]': el(),
  '[data-rule-add]': el(),
  '[data-rule-clear]': el()
};
const root = { querySelector(selector) { return fields[selector] || null; } };
const listeners = {};
const document = {
  querySelector(selector) { return selector === '[data-rule-builder]' ? root : null; },
  getElementById() { return null; },
  querySelectorAll() { return []; },
  addEventListener(name, callback) { (listeners[name] ||= []).push(callback); },
  createElement() { return el(); },
  body: { classList: { add(){}, remove(){} }, appendChild(){} }
};
const js = fs.readFileSync(path.join(__dirname, '../ui/static/app.js'), 'utf8');
vm.runInNewContext(js, { document, window: { addEventListener() {} }, Blob, URL, setTimeout, clearTimeout, setInterval, clearInterval });
for(const callback of listeners.DOMContentLoaded || []) callback();
const add = fields['[data-rule-add]'];
assert.equal(fields['[data-rule-download]'].disabled, true);
ids.id.value = 'CUSTOM_TEST'; ids.title.value = 'Prohibited marker'; ids.pattern.value = 'TODO';
add.click();
let config = JSON.parse(fields['[data-rule-json]'].value);
assert.equal(config.version, 1);
assert.equal(config.rules.length, 1);
assert.equal(config.rules[0].id, 'CUSTOM_TEST');
assert.equal(fields['[data-rule-download]'].disabled, false);
add.click();
assert.equal(JSON.parse(fields['[data-rule-json]'].value).rules.length, 1, 'duplicate ID should not be added');
assert.match(fields['[data-rule-status]'].textContent, /already exists/);
ids.id.value = 'CUSTOM_SECOND'; ids.path.value = '../private';
add.click();
assert.equal(JSON.parse(fields['[data-rule-json]'].value).rules.length, 1, 'unsafe path should not be added');
ids.path.value = 'src/*.go'; add.click();
config = JSON.parse(fields['[data-rule-json]'].value);
assert.equal(config.rules.length, 2);
assert.equal(config.rules[1].path_glob, 'src/*.go');
fields['[data-rule-clear]'].click();
assert.equal(fields['[data-rule-json]'].value, '');
assert.equal(fields['[data-rule-download]'].disabled, true);
console.log('PASS: local UI rule builder add / duplicate / unsafe path / second rule / reset');
