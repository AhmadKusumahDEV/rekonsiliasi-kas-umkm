// Capture only localhost fixture data; never record a live customer workspace.
const fs=require('node:fs/promises'),path=require('node:path'),crypto=require('node:crypto');
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const {setupFixture}=require('./guide-fixture.cjs');
const root=path.resolve(__dirname,'../..'),base=process.env.GUIDE_APP_URL||'http://127.0.0.1:3058';
if(!['localhost','127.0.0.1'].includes(new URL(base).hostname))throw Error('Use the local documentation server');
const output=path.join(root,'docs-site/public'),records=[];
async function waitApp(page,view){await page.goto(base+'/app#'+view);await page.locator(view==='forgot'?'.auth-form':'.cash-header').waitFor();await page.evaluate(()=>document.fonts.ready);await page.waitForTimeout(300)}
async function annotate(page,labels){await page.evaluate(labels=>{
 document.querySelectorAll('.doc-step').forEach(x=>x.remove());
 for(const [i,selector]of labels.entries()){const target=document.querySelector(selector);if(!target)continue;const r=target.getBoundingClientRect();const n=document.createElement('span');n.className='doc-step';n.textContent=String(i+1);n.style.cssText=`position:fixed;z-index:99999;top:${Math.max(4,r.top+4)}px;left:${Math.max(4,r.left+4)}px;width:28px;height:28px;display:grid;place-items:center;background:#f59e0b;color:#102531;border:2px solid white;border-radius:50%;font:600 16px Arial;pointer-events:none`;document.body.append(n)}
 },labels)}
async function capture(browser){
 for(const [id,view,labels]of[
 ['ringkasan','overview',['.cash-account','.cash-totals']],['impor','imports',['#import-sale','#import-receipt','#flex-import button[type=submit]']],['verifikasi','verifikasi',['.exception-card','.exception-card .inline-actions']],['transaksi','entries',['.toolbar','.table-scroll']],['laporan','reports',['[data-action=export]','.metrics-wrap']],['pengaturan','settings',['.details','[data-action=password]']],['reset','forgot',['[name=email]','button[type=submit]']],['kosongkan','settings',['[name=clear_password]','[name=confirm]']]]){
  const context=await browser.newContext({viewport:{width:1440,height:900},reducedMotion:'reduce',serviceWorkers:'block'});const page=await context.newPage();setupFixture(page);await waitApp(page,view);
  if(id==='impor'){for(const kind of ['sale','receipt'])await page.locator('#import-'+kind).setInputFiles(path.join(output,'examples',kind==='sale'?'penjualan.csv':'mutasi-bank.csv'));await page.locator('#flex-import button[type=submit]').click();await page.locator('[data-action=flow-save]').waitFor();}
  if(id==='kosongkan'){await page.getByRole('button',{name:'Kosongkan data workspace',exact:true}).click();}
  if(id==='verifikasi')await page.locator('.exception-card').scrollIntoViewIfNeeded();
  await annotate(page,labels);const file=`images/${id}-1440.png`;await page.screenshot({path:path.join(output,file)});
  const bytes=await fs.readFile(path.join(output,file));records.push({id,file,viewport:{width:1440,height:900},source:'localhost with synthetic API fixtures',sha256:crypto.createHash('sha256').update(bytes).digest('hex'),capturedAt:new Date().toISOString(),labels});await context.close();console.log('Captured '+id);
 }
}
async function record(browser,item){
 const dir=path.join(root,'.cache/guide-recordings',item.id);await fs.mkdir(dir,{recursive:true});
 const context=await browser.newContext({viewport:{width:1280,height:720},recordVideo:{dir,size:{width:1280,height:720}},serviceWorkers:'block',reducedMotion:'reduce'});
 const page=await context.newPage();const flags=setupFixture(page,{empty:item.id!=='verifikasi'});
 await waitApp(page,item.id==='impor'?'imports':item.id==='verifikasi'?'verifikasi':'overview');
 const captionCSS = '.guide-caption{position:fixed;inset:auto 20px 12px;margin:0;width:auto;z-index:2147483647;background:#102531f2;border-left:5px solid #f59e0b;border-radius:8px;color:#fff;padding:14px 20px;font:18px/1.4 Arial;box-shadow:0 5px 20px #0003;pointer-events:none}.guide-caption small{display:block;color:#a5d5ed;font-size:12px;margin-bottom:3px}';
 await page.route(base+'/__guide-caption.css',r=>r.fulfill({status:200,contentType:'text/css',body:captionCSS}));
 await page.evaluate(async url=>{await new Promise(resolve=>{const el=document.createElement('link');el.rel='stylesheet';el.href=url;el.onload=resolve;document.head.append(el)})},base+'/__guide-caption.css');
 const start=Date.now();
 const at=s=>{const [m,sec]=s.split(':').map(Number);return m*60+sec};
 for(let i=0;i<item.scenes.length;i++){
  const scene=item.scenes[i];const delay=at(scene.at)*1000-(Date.now()-start);if(delay>0)await page.waitForTimeout(delay);
  await page.evaluate(({caption,index,total})=>{let el=document.querySelector('.guide-caption');if(!el){el=document.createElement('div');el.className='guide-caption';el.popover='manual';document.body.append(el)}el.replaceChildren();const small=document.createElement('small');small.textContent=`EFA · langkah ${index+1}/${total} · data simulasi`;el.append(small,document.createTextNode(caption));},{caption:scene.caption,index:i,total:item.scenes.length});
  if(item.id==='impor'){
   if(i===1){await page.locator('#import-sale').setInputFiles(path.join(output,'examples',item.id==='catat-harian'?'harian.csv':'penjualan.csv'));await page.locator('#import-receipt').setInputFiles(path.join(output,'examples','mutasi-bank.csv'));}
   if(i===2)await page.locator('#flex-import button[type=submit]').click();
   if(i===3)await page.locator('#flow-preview').scrollIntoViewIfNeeded();
   if(i===4)await page.locator('[data-action=flow-save]').click();
   if(i===5)await page.locator('#flow-preview').scrollIntoViewIfNeeded();
   if(i===6){await page.locator('nav a[href="#verifikasi"]').first().click();await page.locator('.exception-card').first().scrollIntoViewIfNeeded();}
  }else if(item.id==='verifikasi'){
   if(i===1)await page.locator('.exception-card').first().scrollIntoViewIfNeeded();
   if(i===2)await page.getByRole('button',{name:'Periksa dan setujui'}).click();
   if(i===3){await page.locator('[name=amount]').focus();await page.waitForTimeout(700);await page.locator('[name=fee]').focus();}
   if(i===4){await page.locator('[name=reason]').fill('Potongan layanan Rp10.000 sudah diperiksa pada bukti.');await page.locator('#modal button[type=submit]').click();}
   if(i===5)await page.locator('.allocation-history').scrollIntoViewIfNeeded();
  }else{
   if(i===1){await page.locator('[data-action=import]').first().click();await page.locator('#flex-import').waitFor();}
   if(i===2)await page.locator('#import-sale').setInputFiles(path.join(output,'examples',item.id==='catat-harian'?'harian.csv':'penjualan.csv'));
   if(i===3){await page.locator('#flex-import button[type=submit]').click();await page.locator('[data-action=flow-save]').click();}
   if(i===4)await page.locator('nav a[href="#overview"]').first().click();
   if(i===5)await page.locator('nav a[href="#reports"]').first().click();
  }
  await page.evaluate(()=>{const el=document.querySelector('.guide-caption');el.hidePopover();el.showPopover()});
  if(i===0)await page.screenshot({path:path.join(output,'video',item.id+'-poster.png')});
 }
 const remainder=item.duration*1000-(Date.now()-start);if(remainder>0)await page.waitForTimeout(remainder);
 const recording=page.video();await context.close();const file=await recording.path();await fs.copyFile(file,path.join(root,'.cache/guide-recordings',item.id+'.webm'));
 console.log('Recorded '+item.id+' ('+item.duration+' seconds)');
}
(async()=>{await fs.mkdir(path.join(output,'images'),{recursive:true});await fs.mkdir(path.join(output,'video'),{recursive:true});const browser=await chromium.launch({channel:'chrome',headless:true});try{if(!process.argv.includes('--videos-only'))await capture(browser);if(records.length)await fs.writeFile(path.join(__dirname,'panduan-manifest.json'),JSON.stringify({data:'synthetic only',sourceDirty:true,sourceAssets:await Promise.all(['frontend/assets/app.js','frontend/assets/import-flow.js','frontend/assets/verification.js','frontend/assets/efa.css'].map(async file=>({file,sha256:crypto.createHash('sha256').update(await fs.readFile(path.join(root,file))).digest('hex')}))),sourceCommit:require('node:child_process').execFileSync('git',['rev-parse','HEAD'],{cwd:root,encoding:'utf8'}).trim(),records},null,2)+'\n');if(!process.argv.includes('--screenshots-only'))for(const item of JSON.parse(await fs.readFile(path.join(root,'docs-site/video-manifest.json'),'utf8')))if(!process.argv.includes('--verifikasi-only')||item.id==='verifikasi')await record(browser,item);}finally{await browser.close()}})().catch(e=>{console.error(e);process.exit(1)});
