import test from 'node:test';
import assert from 'node:assert/strict';
import {validateUpload,MAX_UPLOAD_BYTES,aiStatusMessage,uploadErrorMessage,allocationOrigin,readableAllocationReason} from '../assets/import-status.js';
test('upload separates 10 MB raw file limit from canonical CSV limits',()=>{
  assert.doesNotThrow(()=>validateUpload({name:'DATA.XLSX',size:MAX_UPLOAD_BYTES}));
  assert.doesNotThrow(()=>validateUpload({name:'data.csv',size:2_000_000}));
  for(const name of ['data.xls','data.pdf','data.xlsx.exe'])assert.throws(()=>validateUpload({name,size:10}),/CSV atau XLSX/);
  assert.throws(()=>validateUpload({name:'data.xlsx',size:MAX_UPLOAD_BYTES+1}),/10 MB/);
  assert.throws(()=>validateUpload({name:'data.csv',size:0}),/File kosong/);
});
test('all AI states describe rule fallback without treating failed AI as a failed import',()=>{
  assert.equal(aiStatusMessage({ai_status:'ok',ai_matched:2}),'2 pasangan dicek AI dan disimpan otomatis');
  for(const status of ['no_candidates','disabled','skipped','failed'])assert.ok(aiStatusMessage({ai_status:status}).length>20);
  assert.match(aiStatusMessage({ai_status:'failed'}),/Hasil aturan tetap tersimpan/);
  assert.match(aiStatusMessage({}),/belum tersedia/);
});
test('HTTP failure guidance includes recovery action without hiding the backend message',()=>{
  for(const status of [400,413,415,422,409,429,503]){
    const text=uploadErrorMessage({status,message:'Baris 8 salah.'});assert.ok(text.startsWith('Baris 8 salah.'));assert.ok(text.length>35);
  }
});
test('automation labels use the backend reason format and hide its internal score',()=>{
  assert.equal(allocationOrigin({reason:'Cocok otomatis (skor 1.00): referensi sama'}),'rules');
  assert.equal(allocationOrigin({reason:'Disarankan AI (skor aturan 0.90): tanggal sesuai'}),'ai');
  assert.equal(allocationOrigin({reason:'Manual, pelanggan menyebut AI'}),null);
  assert.equal(readableAllocationReason('Disarankan AI (skor aturan 0.90): bukti sesuai'),'Dicek AI: bukti sesuai');
});
