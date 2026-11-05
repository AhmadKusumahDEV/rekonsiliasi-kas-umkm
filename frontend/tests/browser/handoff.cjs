const {fixtureRequest}=require('./fixture-request.cjs');
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const assert=require('node:assert/strict'),fs=require('node:fs/promises');
const base=process.env.HANDOFF_URL||'http://127.0.0.1:3058';
if(!['127.0.0.1','localhost'].includes(new URL(base).hostname))throw Error('Use local isolated fixtures only');
const results=[];
(async()=>{
 const browser=await chromium.launch({channel:'chrome',headless:true});
 async function setup(){
  const context=await browser.newContext({serviceWorkers:'block',reducedMotion:'reduce',viewport:{width:1440,height:1000}});
  const page=await context.newPage();page.setDefaultTimeout(7000);const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.addInitScript(()=>window.EventSource=class{constructor(){this.readyState=0}addEventListener(){}close(){}});
  const entry=(id,kind,amount,ref,extra={})=>({id:id.repeat(32),source_id:id,kind,amount,reference:ref,name:'Pemilik Fixture',channel:'qris',date:'2026-11-03',canceled:false,...extra});
  const entries=[entry('a','sale',100000,'UNIQUE'),entry('b','receipt',100000,'UNIQUE'),entry('c','sale',150000,'FEE'),entry('d','receipt',143500,'FEE'),entry('e','sale',70000,'CASH',{category:'cash',settled:true}),entry('f','receipt',2000,'INTEREST',{category:'non_operational'}),entry('1','sale',200000,'AMBIG'),entry('2','receipt',200000,'AMBIG'),entry('3','receipt',200000,'AMBIG')];
  const ex=(entry,extra={})=>({entry,status:'unmatched',remaining:entry.amount,candidate_ids:[],candidate_count:0,suggestion_available:false,reason:'Alasan '+entry.reference,...extra});
  const exceptions=[ex(entries[0],{candidate_ids:[entries[1].id],candidate_count:1,suggestion_available:true}),ex(entries[1],{candidate_ids:[entries[0].id],candidate_count:1,suggestion_available:true}),ex(entries[2],{fee_hint:{amount:6500,source:'bank',label:'Biaya bank',confidence:'tinggi',reason:'Biaya QRIS perlu dikonfirmasi'}}),ex(entries[3]),ex(entries[6],{status:'ambiguous',candidate_ids:[entries[7].id,entries[8].id],candidate_count:22,candidate_ids_truncated:true})];
  const data={workspace:{id:'w',name:'Kontrak Fixture',version:1},state:{entries,imports:[],allocations:[]},events:[],users:[]};
  const flags={loggedIn:true,csrf:'old',password:'Original!2026',data,exceptions,suggestions:[{sale:entries[0],receipt:entries[1],reason:'Unik dari server'}],calls:[],commits:[],allocations:[],failReceipt:false,zeroReceipt:false,failExceptions:false,writeFail:'',forgotFail:0,clearFail:0,holdCommit:false};
  await page.route('**/api/v1/**',async route=>{
   const req=route.request(),url=new URL(req.url()),path=url.pathname,method=req.method(),body=await fixtureRequest(req);flags.calls.push({path,method,body});
   const send=(value,status=200)=>route.fulfill({status,contentType:'application/json',body:JSON.stringify(value)});
   const user={id:'u',workspace_id:'w',name:'Fixture',email:'fixture@example.test',role:'owner'};
   if(path.endsWith('/config'))return send({kind:'umkm',brand:'EFA',limits:{max_import_rows:500,max_csv_bytes:1048576,max_entries:10000}});
   if(path.endsWith('/auth/me'))return flags.loggedIn?send({user,csrf:flags.csrf}):send({message:'Sesi berakhir'},401);
   if(path.endsWith('/auth/login')){if(body.password!==flags.password)return send({message:'Kata sandi salah'},401);flags.loggedIn=true;return send({user,csrf:flags.csrf});}
   if(path.endsWith('/password/forgot'))return flags.forgotFail?send({message:'Terlalu banyak percobaan'},flags.forgotFail):send({ok:true},202);
   if(path.endsWith('/password/reset')){if(body.token!=='a'.repeat(64))return send({message:'Token kedaluwarsa'},400);flags.password=body.password;flags.loggedIn=false;return send({ok:true});}
   if(path.endsWith('/umkm/state'))return send(data);
   if(path.endsWith('/umkm/exceptions'))return flags.failExceptions?send({message:'Antrean tidak tersedia'},503):send(flags.exceptions);
   if(path.endsWith('/umkm/suggestions'))return send(flags.suggestions);
   if(path.endsWith('/umkm/audit'))return send(data.events);
   if(path.endsWith('/imports/files')){
    if(body.invalidUTF8)return send({message:'File harus CSV UTF-8.'},400);
    const zero=body.kind==='receipt'&&flags.zeroReceipt;
    return send({csv:'CANONICAL_'+body.kind,count:zero?0:2,total:zero?0:200000,skipped:0,warnings:[body.kind==='receipt'?'Baris debit dilewati.':'Referensi kosong dinormalkan.'],preview:zero?[]:[entries[0],entries[1]],checksum:body.kind});
   }
   if(path.endsWith('/umkm/imports')){
    flags.commits.push(body);
    if(flags.holdCommit)await new Promise(r=>flags.releaseCommit=r);
    if(body.kind==='receipt'&&flags.failReceipt){flags.failReceipt=false;return send({message:'Sementara gagal'},503);}
    return send({id:'9'.repeat(32),imported:2,skipped:0,auto_matched:body.kind==='receipt'?1:0,ai_matched:0},201);
   }
   if(path.endsWith('/umkm/allocations')){
    flags.allocations.push(body);
    if(flags.writeFail){const code=flags.writeFail;flags.writeFail='';if(code===403)flags.csrf='new';return send({message:code===409?'Saldo berubah':'CSRF'},code);}
    return send({id:'8'.repeat(32)},201);
   }
   if(path.endsWith('/umkm/data')&&method==='DELETE'){
    if(flags.clearFail)return send({message:'Ditolak'},flags.clearFail);
    if(body.password!==flags.password||body.confirm!=='HAPUS')return send({message:'Kata sandi saat ini belum cocok.'},400);
    data.state={entries:[],imports:[],allocations:[]};data.events=[{id:1,action:'Kosongkan data workspace',actor:'Fixture',detail:'Dikosongkan',created_at:'2026-11-04T00:00:00Z',entity_type:null,entity_id:null}];data.workspace.version++;flags.exceptions=[];flags.suggestions=[];return send({ok:true});
   }
   return send({message:'Unknown fixture '+path},404);
  });
  const open=async route=>{await page.goto(base+'/app#'+route);await page.locator(flags.loggedIn&&!['forgot','reset'].includes(route.split('?')[0])?'.cash-header':'.auth-form').waitFor();await page.waitForTimeout(100)};
  const file=(name,text='RAW_FILE')=>({name,mimeType:'text/csv',buffer:Buffer.from(text)});
  return {page,context,flags,open,errors,entries,file};
 }
 async function check(name,fn){const t=await setup();try{await fn(t);assert.deepEqual(t.errors,[]);results.push({name,pass:true})}catch(err){results.push({name,pass:false,error:err.message})}finally{t.flags.releaseCommit?.();await t.context.close()}}
 async function preview(t,kinds=['sale','receipt']){await t.open('imports');for(const kind of kinds)await t.page.locator('#import-'+kind).setInputFiles(t.file(kind+'.csv'));await t.page.locator('#flex-import button[type=submit]').click();await t.page.getByRole('button',{name:'Simpan file yang sudah diperiksa'}).waitFor();}
 await check('two slots preview warnings and commit canonical CSV in sale-first order',async t=>{
  await preview(t);assert.match(await t.page.locator('#flow-preview').innerText(),/Baris debit dilewati/);assert.equal(t.flags.commits.length,0);
  await t.page.locator('[data-action=flow-save]').click();await t.page.getByText('2 baris tersimpan · 1 pasangan otomatis · 0 pasangan AI · 0 dilewati.',{exact:true}).waitFor();
  assert.deepEqual(t.flags.commits,[{kind:'sale',csv:'CANONICAL_sale'},{kind:'receipt',csv:'CANONICAL_receipt'}]);
 });
 await check('one slot is valid; no selected file yields an actionable error',async t=>{
  await t.open('imports');await t.page.locator('#flex-import button[type=submit]').click();await t.page.getByText('Pilih minimal satu file CSV atau XLSX.',{exact:true}).waitFor();
  await t.page.locator('#import-receipt').setInputFiles(t.file('bank.csv'));await t.page.locator('#flex-import button[type=submit]').click();await t.page.locator('[data-action=flow-save]').click();await t.page.getByText('2 baris tersimpan · 1 pasangan otomatis · 0 pasangan AI · 0 dilewati.',{exact:true}).waitFor();assert.equal(t.flags.commits.length,1);assert.equal(t.flags.commits[0].kind,'receipt');
 });
 await check('debit-only preview keeps warnings and never offers an empty commit',async t=>{
  t.flags.zeroReceipt=true;await t.open('imports');await t.page.locator('#import-receipt').setInputFiles(t.file('bank.csv'));await t.page.locator('#flex-import button[type=submit]').click();await t.page.getByText('Baris debit dilewati.',{exact:true}).waitFor();assert.equal(await t.page.locator('[data-action=flow-save]').count(),0);assert.equal(t.flags.commits.length,0);
 });
 await check('partial batch failure retries receipt only; changing a file invalidates its preview',async t=>{
  t.flags.failReceipt=true;await preview(t);await t.page.locator('[data-action=flow-save]').click();await t.page.getByText(/Sementara gagal/).waitFor();assert.deepEqual(t.flags.commits.map(x=>x.kind),['sale','receipt']);
  await t.page.locator('[data-action=flow-save]').click();await t.page.waitForFunction(()=>document.querySelector('#flow-preview').textContent.includes('1 pasangan otomatis'));assert.deepEqual(t.flags.commits.map(x=>x.kind),['sale','receipt','receipt']);
  await t.page.locator('#import-receipt').setInputFiles(t.file('new-bank.csv'));assert.equal(await t.page.locator('[data-action=flow-save]').count(),0);
 });
 await check('double-clicking save does not duplicate an in-flight import',async t=>{
  t.flags.holdCommit=true;await preview(t,['sale']);await t.page.locator('[data-action=flow-save]').click();
  for(let i=0;i<50&&!t.flags.releaseCommit;i++)await t.page.waitForTimeout(20);
  assert.equal(t.flags.commits.length,1);await t.page.locator('[data-action=flow-save]').evaluate(node=>node.click());assert.equal(t.flags.commits.length,1);
  t.flags.holdCommit=false;t.flags.releaseCommit();await t.page.getByText('2 baris tersimpan · 0 pasangan otomatis · 0 pasangan AI · 0 dilewati.',{exact:true}).waitFor();
 });
 await check('unsupported formats fail locally; invalid CSV encoding is reported by the server',async t=>{
  await t.open('imports');await t.page.locator('#import-receipt').setInputFiles(t.file('bank.pdf'));await t.page.locator('#flex-import button[type=submit]').click();await t.page.locator('#flow-preview [role=alert]').waitFor();assert.equal(t.flags.calls.some(x=>x.path.endsWith('/imports/files')),false);
  await t.page.locator('#import-receipt').setInputFiles({name:'bank.csv',mimeType:'text/csv',buffer:Buffer.from([255,254])});await t.page.locator('#flex-import button[type=submit]').click();await t.page.locator('#flow-preview [role=alert]').waitFor();assert.equal(t.flags.calls.filter(x=>x.path.endsWith('/imports/files')).length,1);assert.equal(t.flags.commits.length,0);
 });
 await check('verification groups unique/fee pairs, excludes settled cash and non-operational',async t=>{
  await t.open('verifikasi');await t.page.getByRole('heading',{name:'Verifikasi',exact:true}).waitFor();assert.equal(await t.page.locator('.exception-card').count(),3);assert.equal(await t.page.locator('.exception-card').filter({hasText:'CASH'}).count(),0);assert.equal(await t.page.locator('.exception-card').filter({hasText:'INTEREST'}).count(),0);
  const card=t.page.locator('.exception-card').filter({hasText:'Biaya QRIS perlu dikonfirmasi'});await card.getByRole('button',{name:'Periksa dan setujui'}).click();assert.equal(await t.page.locator('[name=amount]').inputValue(),'143500');assert.equal(await t.page.locator('[name=fee]').inputValue(),'6500');assert.equal(t.flags.allocations.length,0);await t.page.locator('#modal button[type=submit]').click();await t.page.locator('#modal').waitFor({state:'hidden'});assert.equal(t.flags.allocations[0].amount,143500);assert.equal(t.flags.allocations[0].fee,6500);
 });
 await check('aliases use one menu; ambiguous choice is blank and truncation has manual path',async t=>{
  await t.open('reconcile');assert.equal(new URL(t.page.url()).hash,'#verifikasi');assert.equal(await t.page.locator('.cash-subnav a[href="#reconcile"]').count(),0);
  const card=t.page.locator('.exception-card').filter({hasText:'Alasan AMBIG'});await card.getByRole('button',{name:'Pilih kandidat'}).click();assert.equal(await t.page.locator('[name=receipt_id]').inputValue(),'');await t.page.locator('[data-action=close]').first().click();assert.match(await card.innerText(),/22 kandidat/);assert.equal(await card.getByRole('button',{name:'Cari secara manual'}).count(),1);
  await t.page.goto(base+'/app#exceptions');await t.page.getByRole('heading',{name:'Verifikasi',exact:true}).waitFor();assert.equal(new URL(t.page.url()).hash,'#verifikasi');
 });
 await check('a cash-only workspace does not claim it is waiting for a bank receipt',async t=>{
  t.flags.data.state.entries=[t.entries[4]];t.flags.exceptions=[];t.flags.suggestions=[];await t.open('verifikasi');assert.doesNotMatch(await t.page.locator('.content').innerText(),/Catatan belum lengkap|Belum ada bukti penerimaan/);assert.equal(await t.page.locator('.exception-card').count(),0);
 });
 await check('failed queue never claims zero; cash and non-operational have no table actions',async t=>{
  t.flags.failExceptions=true;await t.open('verifikasi');await t.page.getByText('Antrean belum termuat.',{exact:false}).waitFor();await t.page.goto(base+'/app#entries');for(const reference of ['CASH','INTEREST']){const row=t.page.locator('tbody tr').filter({hasText:reference});assert.equal(await row.locator('[data-action=entry-followup]').count(),0);}
 });
 await check('409 and 403 retain allocation draft and idempotency key without replay',async t=>{
  for(const code of [409,403]){t.flags.writeFail=code;await t.open('verifikasi');await t.page.locator('.exception-card').filter({hasText:'Alasan UNIQUE'}).getByRole('button',{name:'Periksa dan setujui'}).click();await t.page.locator('[name=reason]').fill('Draft harus tersimpan');await t.page.locator('#modal button[type=submit]').click();await t.page.locator('[data-form-error]:visible').waitFor();assert.equal(await t.page.locator('[name=reason]').inputValue(),'Draft harus tersimpan');const first=t.flags.allocations.at(-1);await t.page.locator('#modal button[type=submit]').click();await t.page.locator('#modal').waitFor({state:'hidden'});assert.equal(first.key,t.flags.allocations.at(-1).key);}
  assert.equal(t.flags.allocations.length,4);
 });
 await check('forgot is generic; rate limit disables resubmission',async t=>{
  t.flags.loggedIn=false;await t.open('forgot');await t.page.locator('[name=email]').fill('unknown@example.test');await t.page.locator('button[type=submit]').click();await t.page.getByText(/Jika email terdaftar, tautan reset akan dikirim. Periksa/).waitFor();assert.equal(t.flags.calls.filter(x=>x.path.endsWith('/password/forgot')).length,1);
  t.flags.forgotFail=429;await t.page.goto(base+'/app#login');await t.open('forgot');await t.page.locator('[name=email]').fill('known@example.test');await t.page.locator('button[type=submit]').click();await t.page.locator('[data-form-error]:visible').waitFor();assert.equal(await t.page.locator('button[type=submit]').isDisabled(),true);
 });
 await check('reset token stays in memory, disappears from URL, and new password logs in',async t=>{
  t.flags.loggedIn=false;await t.open('reset?token='+'a'.repeat(64));assert.equal(new URL(t.page.url()).hash,'#reset');assert.equal(await t.page.evaluate(()=>localStorage.length),0);
  await t.page.locator('[name=password]').fill('NewPassword!2026');await t.page.locator('[name=confirm_password]').fill('NewPassword!2026');await t.page.locator('button[type=submit]').click();await t.page.locator('[data-form=login]').waitFor();assert.equal(new URL(t.page.url()).hash,'#login');await t.page.locator('[name=email]').fill('fixture@example.test');await t.page.locator('[name=password]').fill('NewPassword!2026');await t.page.locator('button[type=submit]').click();await t.page.locator('.cash-header').waitFor();assert.equal(t.flags.password,'NewPassword!2026');
 });
 await check('invalid reset token and mismatched passwords never send a reset',async t=>{
  t.flags.loggedIn=false;await t.open('reset?token=bad');await t.page.getByText('Tautan reset tidak valid atau tidak lengkap.',{exact:false}).waitFor();assert.equal(await t.page.locator('[data-form=reset]').count(),0);await t.open('reset?token='+'a'.repeat(64));await t.page.locator('[name=password]').fill('NewPassword!2026');await t.page.locator('[name=confirm_password]').fill('Different!2026');await t.page.locator('button[type=submit]').click();await t.page.getByText('Konfirmasi kata sandi belum sama.').waitFor();assert.equal(t.flags.calls.some(x=>x.path.endsWith('/password/reset')),false);
 });
 await check('clear data requires password and exact HAPUS; account and audit remain',async t=>{
  await t.open('settings');await t.page.getByRole('button',{name:'Kosongkan data workspace',exact:true}).click();const submit=t.page.locator('#modal button[type=submit]');assert.equal(await submit.isDisabled(),true);await t.page.locator('[name=clear_password]').fill('WrongPassword!2026');await t.page.locator('[name=confirm]').fill('HAPUS');await submit.click();await t.page.getByText('Kata sandi saat ini belum cocok.',{exact:true}).waitFor();assert.equal(await submit.isDisabled(),true);assert.equal(t.flags.data.state.entries.length,9);
  await t.page.locator('[name=clear_password]').fill('Original!2026');await t.page.locator('[name=confirm]').fill('HAPUS');await submit.click();await t.page.locator('#modal').waitFor({state:'hidden'});assert.equal(new URL(t.page.url()).hash,'#overview');assert.equal(t.flags.data.state.entries.length,0);assert.equal(t.flags.loggedIn,true);await t.page.goto(base+'/app#history');await t.page.locator('.content').getByText('Kosongkan data workspace',{exact:true}).waitFor();
 });
 await check('expired clear-data session cannot replay or leave a destructive dialog active',async t=>{
  t.flags.clearFail=401;await t.open('settings');await t.page.getByRole('button',{name:'Kosongkan data workspace',exact:true}).click();await t.page.locator('[name=clear_password]').fill('Original!2026');await t.page.locator('[name=confirm]').fill('HAPUS');await t.page.locator('#modal button[type=submit]').click();await t.page.locator('[data-form=login]').waitFor();assert.equal(await t.page.locator('#modal').isVisible(),false);assert.equal(t.flags.calls.filter(x=>x.method==='DELETE').length,1);assert.equal(t.flags.data.state.entries.length,9);
 });
 await check('new flows stay within mobile tablet desktop widths',async t=>{
  await fs.mkdir('test-results/handoff',{recursive:true});for(const width of [375,768,1440])for(const route of ['imports','verifikasi','settings']){await t.page.setViewportSize({width,height:900});await t.open(route);assert.equal(await t.page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,width+' '+route);if(width!==768)await t.page.screenshot({path:`test-results/handoff/${route}-${width}.png`,fullPage:true});}
 });
 await browser.close();await fs.mkdir('test-results',{recursive:true});await fs.writeFile('test-results/handoff-browser.json',JSON.stringify(results,null,2));console.log(JSON.stringify(results,null,2));if(results.some(x=>!x.pass))process.exitCode=1;
})().catch(e=>{console.error(e);process.exit(1)});
