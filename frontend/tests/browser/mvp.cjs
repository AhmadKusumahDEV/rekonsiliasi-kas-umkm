const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const assert=require('node:assert/strict'),fs=require('node:fs/promises');
const base=process.env.MVP_URL||'http://127.0.0.1:3058';
if(!['localhost','127.0.0.1'].includes(new URL(base).hostname))throw Error('Fixtures must stay local');
const results=[];
(async()=>{
 const browser=await chromium.launch({channel:'chrome',headless:true});
 async function setup(){
  const context=await browser.newContext({serviceWorkers:'block',reducedMotion:'reduce',viewport:{width:1440,height:1000}});
  const page=await context.newPage();page.setDefaultTimeout(8000);
  const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.addInitScript(()=>{window.__streams=[];window.EventSource=class{constructor(){this.readyState=0;this.events={};window.__streams.push(this);queueMicrotask(()=>this.emit('open'));}addEventListener(n,f){(this.events[n]??=[]).push(f)}emit(n){this.readyState=n==='open'?1:0;for(const f of this.events[n]||[])f({});}close(){this.readyState=2}}});
  const entry=(id,kind,amount,reference)=>({id:id.repeat(32),source_id:id,kind,amount,reference,name:'Ayu',channel:'transfer',date:'2026-11-01',note:'',canceled:false});
  const entries=[entry('a','sale',150000,'UNIQUE'),entry('b','receipt',150000,'UNIQUE'),entry('c','sale',200000,'PARTIAL'),entry('d','receipt',50000,'PARTIAL'),entry('e','receipt',200000,'AMBIG'),entry('f','receipt',200000,'AMBIG'),entry('1','sale',200000,'AMBIG'),entry('2','sale',70000,'UNMATCHED')];
  const allocation={id:'9'.repeat(32),sale_id:entries[2].id,receipt_id:entries[3].id,amount:50000,fee:0,reason:'Cicilan pertama',reversed:false,created_at:'2026-11-01T00:00:00Z'};
  const suggestions=[{sale:entries[0],receipt:entries[1],reason:'Unik dua arah dari fixture'}];
  const ex=(entry,status,remaining,extra={})=>({entry,status,remaining,candidate_ids:[],candidate_count:0,candidate_ids_truncated:false,suggestion_available:false,reason:'Alasan fixture '+entry.reference,...extra});
  const exceptions=[ex(entries[1],'unmatched',150000,{suggestion_available:true,candidate_ids:[entries[0].id],candidate_count:1}),ex(entries[0],'unmatched',150000,{suggestion_available:true,candidate_ids:[entries[1].id],candidate_count:1}),ex(entries[2],'partial',150000),ex(entries[6],'ambiguous',200000,{candidate_ids:[entries[4].id,entries[5].id],candidate_count:25,candidate_ids_truncated:true}),ex(entries[7],'unmatched',70000)];
  const data={workspace:{id:'w',name:'MVP Fixture',version:1},state:{entries,allocations:[allocation],imports:[{id:'8'.repeat(32),kind:'sale',rows:4,canceled:false,created_at:'2026-11-01T00:00:00Z'}]},events:[],users:[]};
  const flags={data,exceptions,suggestions,limits:{max_import_rows:2,max_csv_bytes:1000,max_entries:10},failExceptions:false,failSuggestions:false,noLimits:false,loggedIn:true,csrf:'old-csrf',writeFailure:'',calls:[],writes:[]};
  const events=Array.from({length:130},(_,i)=>({id:200-i,actor:'Ayu',action:'Keputusan '+(200-i),detail:'Catatan fixture',created_at:'2026-11-01T00:00:00Z',entity_type:i===0?'allocation':i===1?'import':null,entity_id:i===0?allocation.id:i===1?'8'.repeat(32):null}));
  await page.route('**/api/v1/**',async r=>{
   const req=r.request(),url=new URL(req.url()),path=url.pathname;flags.calls.push(url.pathname+url.search);
   const send=(body,status=200)=>r.fulfill({status,contentType:'application/json',body:JSON.stringify(body)});
   if(path.endsWith('/config'))return send({kind:'umkm',brand:'EFA',...(flags.noLimits?{}:{limits:flags.limits})});
   if(path.endsWith('/auth/me'))return send({user:{id:'u',workspace_id:'w',name:'Ayu',email:'synthetic@example.test',role:'owner'},csrf:flags.csrf});
   if(path.endsWith('/umkm/state'))return send(data);
   if(path.endsWith('/umkm/suggestions'))return send(flags.failSuggestions?{message:'Unavailable'}:suggestions,flags.failSuggestions?503:200);
   if(path.endsWith('/umkm/exceptions'))return send(flags.failExceptions?{message:'Unavailable'}:exceptions,flags.failExceptions?503:200);
   if(path.endsWith('/umkm/audit')){const before=Number(url.searchParams.get('before_id')||201);return send(events.filter(e=>e.id<before).slice(0,50));}
   if(path.endsWith('/imports/preview'))return send({rows:[entry('3','sale',1,'NEW1'),entry('4','sale',1,'NEW2'),entry('5','sale',1,'NEW3')],count:3,skipped:0,total:3,checksum:'x'});
   if(path.endsWith('/umkm/allocations')){
    flags.writes.push({body:req.postDataJSON(),csrf:req.headers()['x-csrf-token']});
    if(flags.writeFailure==='network'){flags.writeFailure='';return r.abort('failed');}
    if(flags.writeFailure==='csrf'){flags.writeFailure='';flags.csrf='new-csrf';return send({message:'CSRF'},403);}
    if(flags.writeFailure==='conflict'){flags.writeFailure='';return send({message:'Saldo berubah'},409);}
    return send({id:'7'.repeat(32)},201);
   }
   return send({message:'Unknown fixture '+path},404);
  });
  const open=async(view='exceptions')=>{await page.goto(base+'/app#'+view);await page.locator('.cash-header').waitFor();await page.waitForTimeout(120)};
  return {page,context,flags,open,errors,entries};
 }
 async function check(name,fn){const t=await setup();try{await fn(t);assert.deepEqual(t.errors,[]);results.push({name,pass:true})}catch(e){results.push({name,pass:false,error:e.message})}finally{await t.context.close()}}
 await check('reciprocal unique sides show one card and both candidates before opening',async t=>{
  await t.open();assert.equal(await t.page.locator('.exception-card').count(),4);
  const pair=t.page.locator('.exception-card').filter({hasText:'Usulan pasangan unik'});
  assert.equal(await pair.locator('.exception-entry').count(),2);
  assert.match(await pair.innerText(),/Penjualan/);assert.match(await pair.innerText(),/Penerimaan/);
  assert.match(await pair.innerText(),/150.000/);
  assert.equal(await pair.getByRole('button',{name:'Konfirmasi usulan',exact:true}).count(),1);
  await pair.getByRole('button',{name:'Konfirmasi usulan',exact:true}).click();
  await t.page.locator('[data-action=close]').first().click();assert.equal(t.flags.writes.length,0);
 });
 await check('transaction follow-up pins the selected side without choosing its counterpart',async t=>{
  await t.open('entries');
  for(const [entry,field,other] of [[t.entries[0],'sale_id','receipt_id'],[t.entries[1],'receipt_id','sale_id']]) {
   await t.page.locator('[data-action=entry-followup][data-id="'+entry.id+'"]').click();
   assert.equal(await t.page.locator('[name='+field+']').inputValue(),entry.id);
   assert.equal(await t.page.locator('[name='+other+']').inputValue(),'');
   assert.equal(await t.page.locator('[name=amount]').inputValue(),'');
   await t.page.locator('[data-action=close]').first().click();
  }
  const settled=t.page.locator('tbody tr').filter({hasText:'PARTIAL'}).filter({hasText:'Penerimaan'});
  assert.equal(await settled.locator('[data-action=entry-followup]').count(),0);
  assert.match(await settled.innerText(),/Sudah cocok/);assert.equal(t.flags.writes.length,0);
 });
 await check('legacy history and unallocated sales have a plain-language explanation',async t=>{
  await t.open('history');await t.page.getByText('Event lama belum menyimpan tautan ke impor atau alokasi.').first().waitFor();
  await t.page.goto(base+'/app#reports');await t.page.getByText('Penjualan belum teralokasi',{exact:true}).waitFor();
  assert.match(await t.page.locator('.content').innerText(),/bukan kerugian/);assert.match(await t.page.locator('.content').innerText(),/bukan perhitungan laba/);
 });
 await check('SSE failure stays limited through REST success and route changes; reconnect refreshes version',async t=>{
  await t.open();await t.page.getByText('Data terhubung',{exact:true}).waitFor();
  const before=t.flags.calls.filter(x=>x.endsWith('/umkm/state')).length;
  await t.page.evaluate(()=>window.__streams.at(-1).emit('error'));
  await t.page.getByText('Koneksi terbatas',{exact:true}).waitFor();
  await t.page.goto(base+'/app#entries');await t.page.getByText('Koneksi terbatas',{exact:true}).waitFor();
  await t.page.waitForTimeout(15500);
  assert.ok(t.flags.calls.filter(x=>x.endsWith('/umkm/state')).length>before+1);
  assert.equal(await t.page.getByText('Data terhubung',{exact:true}).count(),0);
  t.flags.data.workspace.version=2;
  await t.page.evaluate(()=>{document.activeElement?.blur();window.__streams.at(-1).emit('open')});
  await t.page.getByText('Data terhubung',{exact:true}).waitFor();
  assert.ok(t.flags.calls.filter(x=>x.endsWith('/umkm/state')).length>before+2);
 });
 await check('server byte and row limits are honored',async t=>{
  t.flags.limits.max_csv_bytes=24;await t.open('imports');assert.match(await t.page.locator('.content').innerText(),/24 byte/);
  await t.page.locator('[data-action=import]').first().click();
  await t.page.locator('input[type=file]').setInputFiles({name:'x.csv',mimeType:'text/csv',buffer:Buffer.alloc(25,65)});
  await t.page.locator('#modal button[type=submit]').click();await t.page.locator('[data-form-error]:visible').waitFor();
  assert.equal(t.flags.calls.some(x=>x.includes('/imports/preview')),false);
  await t.page.locator('input[type=file]').setInputFiles({name:'x.csv',mimeType:'text/csv',buffer:Buffer.from('valid')});
  await t.page.locator('#modal button[type=submit]').click();await t.page.getByText('CSV melebihi batas 2 baris data.').waitFor();
  assert.equal(await t.page.locator('[data-action=commit-import]').count(),0);
 });
 await check('missing limits do not fall back to hardcoded policy',async t=>{
  t.flags.noLimits=true;await t.open('imports');await t.page.locator('[data-action=import]').first().click();
  await t.page.locator('#toast[data-open=true]').waitFor();assert.equal(await t.page.locator('#modal').isVisible(),false);
  assert.equal(t.flags.calls.some(x=>x.includes('/imports/preview')),false);
 });
 await check('missing unique suggestion refreshes rather than guessing a pair',async t=>{
  t.flags.failSuggestions=true;await t.open();await t.page.getByRole('button',{name:'Periksa usulan terbaru'}).first().click();
  await t.page.locator('#toast[data-open=true]').waitFor();assert.equal(await t.page.locator('#modal').isVisible(),false);assert.equal(t.flags.writes.length,0);
 });
 await check('403 updates CSRF and requires a deliberate retry with the same key',async t=>{
  t.flags.writeFailure='csrf';await t.open('reconcile');await t.page.getByRole('button',{name:'Periksa pasangan',exact:true}).click();
  await t.page.locator('#modal button[type=submit]').click();await t.page.locator('[data-form-error]:visible').waitFor();assert.equal(t.flags.writes.length,1);
  await t.page.locator('#modal button[type=submit]').click();await t.page.locator('#modal').waitFor({state:'hidden'});
  assert.equal(t.flags.writes[0].csrf,'old-csrf');assert.equal(t.flags.writes[1].csrf,'new-csrf');assert.equal(t.flags.writes[0].body.key,t.flags.writes[1].body.key);
 });
 await check('ambiguous candidates never default to a first match',async t=>{
  await t.open();const card=t.page.locator('.exception-card').filter({hasText:'Alasan fixture AMBIG'});
  await card.getByRole('button',{name:'Pilih kandidat',exact:true}).click();
  assert.equal(await t.page.locator('[name=receipt_id]').inputValue(),'');
  assert.equal(await t.page.locator('[name=receipt_id] option').count(),3);assert.equal(await card.locator('.exception-candidates .exception-entry').count(),2);
  assert.equal(t.flags.writes.length,0);
  await t.page.locator('[data-action=close]').first().click();
  assert.match(await card.innerText(),/25 kandidat/);assert.equal(await card.getByRole('button',{name:'Cari secara manual'}).count(),1);
 });
 await check('partial remains partial; unique reason is visible before saving',async t=>{
  await t.open();await t.page.getByText('Alasan fixture PARTIAL',{exact:true}).waitFor();
  const partial=t.page.locator('.exception-card').filter({hasText:'Alasan fixture PARTIAL'});assert.match(await partial.innerText(),/150.000/);
  await t.page.getByRole('button',{name:'Konfirmasi usulan',exact:true}).click();await t.page.getByText('Unik dua arah dari fixture',{exact:true}).waitFor();assert.equal(t.flags.writes.length,0);
 });
 await check('failed exception request never claims an empty queue',async t=>{
  t.flags.failExceptions=true;await t.open();await t.page.getByText('Antrean belum termuat.',{exact:false}).waitFor();assert.equal(await t.page.getByText('Tidak ada transaksi pada antrean ini',{exact:true}).count(),0);
  await t.page.goto(base+'/app#entries');await t.page.getByText('UNMATCHED',{exact:true}).waitFor();
 });
 await check('130 audit entries load by server cursor and linked entities open',async t=>{
  await t.open('history');await t.page.getByRole('button',{name:'Muat aktivitas lebih lama'}).waitFor();
  await t.page.locator('[data-action=audit-entity]').first().click();await t.page.getByRole('heading',{name:'Detail keputusan terkait'}).waitFor();await t.page.locator('[data-action=close]').first().click();
  await t.page.getByRole('button',{name:'Muat aktivitas lebih lama'}).click();await t.page.waitForFunction(()=>document.querySelector('#list-results').textContent.includes('100'));
  await t.page.getByRole('button',{name:'Muat aktivitas lebih lama'}).click();await t.page.getByText('Seluruh riwayat sudah dimuat.').waitFor();
  assert.ok(t.flags.calls.some(x=>x.includes('before_id=151')));assert.ok(t.flags.calls.some(x=>x.includes('before_id=101')));
  assert.equal(await t.page.locator('[data-action=audit-entity]').count(),2);
 });
 await check('network retry retains the same allocation key and payload',async t=>{
  t.flags.writeFailure='network';await t.open('reconcile');await t.page.getByRole('button',{name:'Periksa pasangan',exact:true}).click();
  await t.page.locator('#modal button[type=submit]').click();await t.page.locator('[data-form-error]:visible').waitFor();
  await t.page.locator('#modal button[type=submit]').click();await t.page.locator('#modal').waitFor({state:'hidden'});
  assert.equal(t.flags.writes.length,2);assert.deepEqual(t.flags.writes[0].body,t.flags.writes[1].body);
 });
 await check('409 refreshes without discarding draft or resubmitting',async t=>{
  t.flags.writeFailure='conflict';await t.open('reconcile');await t.page.getByRole('button',{name:'Periksa pasangan',exact:true}).click();
  await t.page.locator('[name=reason]').fill('Catatan yang harus tetap ada');await t.page.locator('#modal button[type=submit]').click();
  await t.page.getByText('Saldo berubah',{exact:true}).waitFor();assert.equal(await t.page.locator('[name=reason]').inputValue(),'Catatan yang harus tetap ada');assert.equal(t.flags.writes.length,1);
 });
 await check('exceptions and audit stay responsive on phone tablet desktop',async t=>{
  await t.open();for(const width of [375,768,1440])for(const view of ['exceptions','history']){
   await t.page.setViewportSize({width,height:900});await t.page.goto(base+'/app#'+view);await t.page.locator('.content h1').waitFor();assert.equal(await t.page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
   if(view==='history')await t.page.getByRole('button',{name:'Muat aktivitas lebih lama'}).waitFor();
   await t.page.evaluate(()=>document.activeElement?.blur());
   await fs.mkdir('test-results',{recursive:true});
   if(width===1440)await t.page.screenshot({path:'test-results/mvp-synthetic-desktop-'+view+'.png',fullPage:true});
   if(width===375&&view==='exceptions')await t.page.screenshot({path:'test-results/mvp-synthetic-mobile-exceptions.png',fullPage:true});
  }
 });
 await browser.close();await fs.mkdir('test-results',{recursive:true});await fs.writeFile('test-results/mvp-browser.json',JSON.stringify(results,null,2));console.log(JSON.stringify(results,null,2));if(results.some(x=>!x.pass))process.exitCode=1;
})().catch(e=>{console.error(e);process.exit(1)});
