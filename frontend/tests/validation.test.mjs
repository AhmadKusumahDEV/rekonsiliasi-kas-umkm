import test from "node:test";
import assert from "node:assert/strict";
import { validatePassword, deduplicatePreview, selectAllocation } from "../assets/validation.js";

test("password checks count code points and enforce bcrypt's UTF-8 byte ceiling", () => {
  assert.doesNotThrow(() => validatePassword("a".repeat(72)));
  assert.doesNotThrow(() => validatePassword("漢".repeat(24)));
  assert.throws(() => validatePassword("漢".repeat(25)), /72 byte/);
  assert.throws(() => validatePassword("😀".repeat(6)), /12 karakter/);
  assert.doesNotThrow(() => validatePassword("😀".repeat(12)));
});
test("preview totals exclude identical existing entries and reject conflicting IDs", () => {
  const old = { id:"old",source_id:"P101",kind:"sale",date:"2026-10-28",reference:"P101",name:"Ayu",amount:150000,channel:"transfer",note:"" };
  const fresh = { ...old,id:"new",source_id:"S2",amount:50000 };
  const result = deduplicatePreview({rows:[old,fresh]},[old],"sale");
  assert.equal(result.count,1);
  assert.equal(result.skipped,1);
  assert.equal(result.total,50000n);
  assert.throws(() => deduplicatePreview({rows:[{...old,amount:999}]},[old],"sale"),/data berbeda/);
  assert.throws(() => deduplicatePreview({rows:[old]},[{...old,canceled:true}],"sale"),/dibatalkan/);
});
test("a missing explicit allocation target never falls back to another sale", () => {
  const entries=[{id:"s",kind:"sale",amount:20},{id:"r",kind:"receipt",amount:20}];
  assert.throws(()=>selectAllocation(entries,x=>x.amount,"missing","r"),/berubah/);
  assert.equal(selectAllocation(entries,x=>x.amount).sale.id,"s");
});
