import test from 'node:test';
import assert from 'node:assert/strict';
import {importLimits, validateImportSize, uniqueSuggestion, auditTarget, mergeAudit} from '../assets/mvp.js';

test('limits come from the server and missing limits never fall back silently', () => {
  const config={limits:{max_import_rows:2,max_csv_bytes:24,max_entries:5}};
  assert.equal(importLimits(config).max_import_rows,2);
  assert.throws(()=>importLimits({}),/belum tersedia/);
  assert.throws(()=>validateImportSize({size:25},config),/24/);
  assert.equal(validateImportSize({size:24},config).max_entries,5);
});
test('only a flagged unique suggestion can open confirmation from exceptions', () => {
  const matches=[{sale:{id:'s'},receipt:{id:'r'}}];
  assert.equal(uniqueSuggestion({entry:{id:'s'},status:'unmatched',suggestion_available:true},matches),matches[0]);
  for(const status of ['partial','ambiguous']) assert.equal(uniqueSuggestion({entry:{id:'s'},status,suggestion_available:true},matches),null);
  assert.equal(uniqueSuggestion({entry:{id:'s'},status:'unmatched',suggestion_available:false},matches),null);
  assert.equal(uniqueSuggestion({entry:{id:'missing'},suggestion_available:true},matches),null);
});
test('audit links use structured entity fields and cursor pages deduplicate IDs', () => {
  const id='a'.repeat(32);
  assert.equal(auditTarget({entity_type:'allocation',entity_id:id}),'allocation:'+id);
  assert.equal(auditTarget({entity_type:null,entity_id:null,detail:id}),null);
  assert.equal(auditTarget({entity_type:'allocation',entity_id:'../admin'}),null);
  assert.deepEqual(mergeAudit([{id:3}],[{id:3},{id:2},{id:2}]),[{id:3},{id:2}]);
});
