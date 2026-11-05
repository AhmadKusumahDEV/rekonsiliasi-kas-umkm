const {fixtureRequest}=require('./fixture-request.cjs');
const {chromium}=require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const fs=require('node:fs/promises'),assert=require('node:assert/strict');
const base=process.env.AUDIT_URL || 'http://127.0.0.1:3058',results=[];
if (!['localhost','127.0.0.1'].includes(new URL(base).hostname)) throw new Error('Audit fixtures must run against a local static server.');
const sleep=n=>new Promise(r=>setTimeout(r,n));
(async()=>{
 const browser=await chromium.launch({channel:'chrome',headless:true});
 async function setup(route='reconcile',loggedIn=true){
  const context=await browser.newContext({serviceWorkers:'block',viewport:{width:1440,height:1000},reducedMotion:'reduce'});
  const page=await context.newPage();page.setDefaultTimeout(8000);
  if(process.env.AUDIT_BASELINE){const old=require('node:child_process').execFileSync('git',['show','c3a4b67:frontend/assets/app.js'],{encoding:'utf8'});await page.route('**/assets/app.js*',r=>r.fulfill({contentType:'text/javascript',body:old}));} await page.addInitScript(()=>{window.__streams=[];window.EventSource=class{constructor(){this.events={};this.readyState=1;window.__streams.push(this);queueMicrotask(()=>this.emit('open',{}));}addEventListener(n,f){(this.events[n]??=[]).push(f)}emit(n,e){if(this.readyState===1)for(const f of this.events[n]||[])f(e)}close(){this.readyState=2}};window.__emit=v=>window.__streams.forEach(s=>s.emit('state',{data:String(v)}));});
  const sale={id:'a'.repeat(32),source_id:'P101',kind:'sale',date:'2026-10-28',reference:'INV-1',name:'Ayu',amount:150000,channel:'transfer',note:'',canceled:false,created_at:'2026-10-28T00:00:00Z'};
  const receipt={...sale,id:'b'.repeat(32),source_id:'R101',kind:'receipt'};
  const data={workspace:{id:'w',name:'Fixture Audit',version:1},state:{entries:[sale,receipt],imports:[],allocations:[]},users:[],events:[]};
  const user={id:'u',workspace_id:'w',name:'Pemilik Uji',email:'fixture@example.test',role:'owner'};
  const fixture={data,sale,receipt,loggedIn,holdNext:false,stateHeld:false,release:null,exportExpired:false,holdPreview:false,previewHeld:false,previewRelease:null,calls:[]};
  await page.route('**/api/v1/**',async r=>{
   const req=r.request(),path=new URL(req.url()).pathname,method=req.method();fixture.calls.push({path,method});
   const send=(body,status=200,type='application/json')=>r.fulfill({status,contentType:type,body:typeof body==='string'?body:JSON.stringify(body)});
   if(path.endsWith('/config'))return send({brand:'Easy Financial Audit',kind:'umkm',limits:{max_import_rows:500,max_csv_bytes:1048576,max_entries:10000}});
   if(path.endsWith('/umkm/exceptions'))return send(data.state.allocations.length?[]:[sale,receipt].map(entry=>({entry,remaining:entry.amount,status:'unmatched',candidate_ids:[entry.id===sale.id?receipt.id:sale.id],candidate_count:1,suggestion_available:true,reason:'Fixture'})));
   if(path.endsWith('/auth/me'))return send(fixture.loggedIn?{user,csrf:'fake-csrf'}:{message:'Sesi berakhir'},fixture.loggedIn?200:401);
   if(path.endsWith('/auth/login')){fixture.loggedIn=true;return send({user,csrf:'fake-csrf'});}
   if(path.endsWith('/umkm/state')){
    if(fixture.failState)return send({message:'Fixture unavailable'},503);
    const snapshot=structuredClone(data);
    if(fixture.holdNext){fixture.holdNext=false;fixture.stateHeld=true;await new Promise(resolve=>fixture.release=resolve);}
    return send(snapshot);
   }
   if(path.endsWith('/umkm/suggestions'))return fixture.failSuggestions?send({message:'Fixture suggestions unavailable'},503):send(fixture.suggestionsOverride|| (data.state.allocations.length?[]:[{sale,receipt,reason:'Fixture'}]));
   if(path.endsWith('/imports/files')){
    const body=await fixtureRequest(req);if(body.invalidUTF8)return send({message:'File harus CSV UTF-8.'},400);
    if(fixture.holdPreview){fixture.previewHeld=true;await new Promise(resolve=>fixture.previewRelease=resolve);}
    return send({csv:'CANONICAL',preview:[{...sale,id:'c'.repeat(32),source_id:'S2',reference:'INV-2',amount:50000}],count:1,total:50000,skipped:1,warnings:[],checksum:'test'});
   }
   if(path.endsWith('/umkm/allocations')&&method==='POST'){
    const body=req.postDataJSON();data.workspace.version=2;data.state.allocations.push({...body,id:body.key,reversed:false,created_at:'2026-10-28T10:00:00Z'});
    await page.evaluate(()=>window.__emit(2));return send(data.state.allocations[0],201);
   }
   if(path.endsWith('/umkm/export')){fixture.loggedIn=false;return send({message:'Sesi berakhir'},401);}
   return send({message:'Unsupported fixture '+path},404);
  });
  await page.goto(base+'/app#'+route,{waitUntil:'domcontentloaded'});
  await page.locator(loggedIn?'.cash-header':'form[data-form=login]').waitFor();
  await sleep(120);
  return {context,page,fixture};
 }
 async function check(name,fn){let t;try{t=await setup();await fn(t);results.push({name,pass:true});}catch(error){results.push({name,pass:false,error:error.message.split('\n')[0]});}finally{t?.fixture.release?.();t?.fixture.previewRelease?.();await sleep(100);await t?.context.close();}}
 await check('active session auth route',async({page})=>{await page.goto(base+'/app#login');await page.locator('.content h1').waitFor();assert.doesNotMatch(await page.locator('.content h1').innerText(),/tidak tersedia/);});
 await check('dialog initial focus',async({page})=>{await page.goto(base+'/app#overview',{waitUntil:'domcontentloaded'});await page.locator('[data-action=allocate]').first().click();assert.notEqual(await page.evaluate(()=>document.activeElement.dataset.action),'close');});
 await check('dialog outside click',async({page})=>{await page.goto(base+'/app#overview',{waitUntil:'domcontentloaded'});await page.locator('[data-action=allocate]').first().click();await page.mouse.click(5,5);await page.locator('#modal').waitFor({state:'hidden'});});
 async function previewSetup(t){await t.page.goto(base+'/app#imports');await t.page.locator('#import-sale').setInputFiles({name:'test.csv',mimeType:'text/csv',buffer:Buffer.from('VALID')});}
 await check('preview duplicate total',async t=>{await previewSetup(t);await t.page.locator('#flex-import button[type=submit]').click();await t.page.getByRole('button',{name:'Simpan file yang sudah diperiksa'}).waitFor();const text=await t.page.locator('#flow-preview').innerText();assert.match(text,/50.000/);assert.doesNotMatch(text,/200.000/);});
 await check('preview action label',async t=>{await previewSetup(t);t.fixture.holdPreview=true;await t.page.locator('#flex-import button[type=submit]').click();await t.page.waitForFunction(()=>document.querySelector('#flex-import').dataset.busy==='true');assert.match(await t.page.locator('#flex-import button[type=submit]').innerText(),/Memeriksa/);t.fixture.previewRelease?.();});
 await check('missing allocation target',async({page})=>{await page.locator('[data-action=allocate]').first().evaluate(el=>el.dataset.id='missing');await page.locator('[data-action=allocate]').first().click();assert.equal(await page.locator('#modal').isVisible(),false);});
 await check('fresh state after save while another refresh runs',async t=>{
  t.fixture.holdNext=true;await t.page.evaluate(()=>window.__emit(2));
  for(let i=0;i<50&&!t.fixture.stateHeld;i++)await sleep(20);
  assert.equal(t.fixture.stateHeld,true);
  await t.page.locator('[data-action=allocate]').first().click();
  await t.page.locator('[name=sale_id]').selectOption(t.fixture.sale.id);await t.page.locator('[name=receipt_id]').selectOption(t.fixture.receipt.id);await t.page.locator('[name=amount]').fill('150000');await t.page.locator('#modal button[type=submit]').click();
  await t.page.locator('#modal').waitFor({state:'hidden'});
  t.fixture.release();await t.page.getByRole('button',{name:'Batalkan pasangan'}).waitFor();
 });
 await check('expired export stays in app',async({page})=>{await page.goto(base+'/app#reports');await page.locator('a[href$="/umkm/export"]').click();await page.locator('form[data-form=login]').waitFor();assert.equal(new URL(page.url()).pathname,'/app');});
 if(!process.env.AUDIT_BASELINE){
  await check('short mobile menu keeps close control reachable',async({page})=>{
   await page.setViewportSize({width:375,height:320});
   await page.getByRole('button',{name:'Menu',exact:true}).click();
   await page.getByRole('button',{name:'Tutup menu',exact:true}).click();
   assert.equal(await page.locator('#mobile-menu').isVisible(),false);
   assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  });
  await check('suggestion failure preserves newly fetched entries',async({page,fixture})=>{
   fixture.failSuggestions=true;fixture.data.workspace.version=2;
   fixture.data.state.entries.push({...fixture.sale,id:'d'.repeat(32),source_id:'NEW',reference:'INV-NEW-STATE'});
   await page.goto(base+'/app#entries');await page.reload();await page.getByText('INV-NEW-STATE',{exact:true}).waitFor();
  });
  await check('stale suggestion cannot become an allocation',async({page,fixture})=>{
   fixture.suggestionsOverride=[{sale:{...fixture.sale,id:'missing'},receipt:fixture.receipt,reason:'Old snapshot'}];
   await page.goto(base+'/app#reconcile');await page.reload();await page.locator('.content h1').waitFor();
   assert.equal(await page.locator('[data-action=match]').count(),0);
  });
  await check('invalid UTF-8 CSV rejected by upload API without committing',async t=>{
   await previewSetup(t);await t.page.locator('#import-sale').setInputFiles({name:'broken.csv',mimeType:'text/csv',buffer:Buffer.from([0xff,0xfe,0x41])});await t.page.locator('#flex-import button[type=submit]').click();await t.page.locator('#flow-preview [role=alert]').waitFor();assert.equal(t.fixture.calls.filter(c=>c.path.endsWith('/imports/files')).length,1);
  });
  await check('authenticated data failure offers recovery instead of login',async({page,fixture})=>{
   fixture.failState=true;await page.goto(base+'/app#overview');await page.reload();
   await page.getByRole('heading',{name:'Sudah masuk, catatan belum terhubung'}).waitFor();
   assert.equal(await page.locator('form[data-form=login]').count(),0);
   fixture.failState=false;await page.getByRole('button',{name:'Muat catatan'}).click();
   await page.locator('.cash-header').waitFor();
  });
 }
 await browser.close();await fs.mkdir('test-results',{recursive:true});await fs.writeFile('test-results/browser-'+(process.env.AUDIT_BASELINE?'before':'after')+'.json',JSON.stringify(results,null,2));console.log(JSON.stringify(results,null,2));if(results.some(r=>!r.pass)&&!process.env.AUDIT_BASELINE)process.exitCode=1;
})().catch(e=>{console.error(e);process.exit(1)});
