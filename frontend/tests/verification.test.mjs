import test from 'node:test';import assert from 'node:assert/strict';
import {allocatable,verificationRows,appRoute} from '../assets/verification.js';
const entry=(id,kind,amount,extra={})=>({id,kind,amount,reference:'INV-1',...extra});
const ex=(entry,remaining=entry.amount,extra={})=>({entry,remaining,status:'unmatched',candidate_ids:[],...extra});
test('settled cash and non-operational entries never become allocation choices',()=>{
 for(const e of [entry('a','sale',10,{settled:true}),entry('a','sale',10,{category:'cash'}),entry('a','receipt',10,{category:'non_operational'}),entry('a','sale',10,{canceled:true})])assert.equal(allocatable(e),false);
});
test('unique suggestions group two sides and keep backend remaining values',()=>{
 const sale=entry('s','sale',100),receipt=entry('r','receipt',100),items=[ex(sale,70),ex(receipt,70)];
 const rows=verificationRows(items,[{sale,receipt,reason:'Server'}]);assert.equal(rows.length,1);assert.equal(rows[0].items[0].remaining,70);assert.equal(rows[0].category,'ready');
});
test('fee hints fill a unique reciprocal candidate, never guessing among duplicates',()=>{
 const sale=entry('s','sale',150000),receipt=entry('r','receipt',143500),items=[ex(sale,150000,{fee_hint:{amount:6500}}),ex(receipt)];
 assert.equal(verificationRows(items,[])[0].match.fee,6500);
 const duplicate=entry('other','receipt',143500);assert.equal(verificationRows([...items,ex(duplicate)],[]).some(x=>x.match),false);
 const anotherSale=entry('s2','sale',150000);assert.equal(verificationRows([...items,ex(anotherSale,150000,{fee_hint:{amount:6500}})],[]).some(x=>x.match),false);
});
test('reciprocal single-candidate exceptions share a card without becoming approved',()=>{
 const items=[ex(entry('s','sale',100),100,{status:'ambiguous',candidate_ids:['r']}),ex(entry('r','receipt',100),100,{status:'ambiguous',candidate_ids:['s']})];
 const rows=verificationRows(items,[]);assert.equal(rows.length,1);assert.equal(rows[0].items.length,2);assert.equal(rows[0].match,null);
});
test('legacy routes alias to verification without exposing a reset token as a route',()=>{
 assert.equal(appRoute('#reconcile'),'verifikasi');assert.equal(appRoute('#exceptions'),'verifikasi');assert.equal(appRoute('#reset?token=secret'),'reset');
});
