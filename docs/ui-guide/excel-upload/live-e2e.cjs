// Only static branch assets are overridden. API requests go to the real backend.
const fs=require('node:fs/promises'),path=require('node:path'),assert=require('node:assert/strict');
const {chromium,request}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const root=path.resolve(__dirname,'../../..'),origin='https://efa.madgo.my.id',api='https://temukas.madgo.my.id';
const fixtureDir=path.join(__dirname,'fixtures'),shotDir=path.join(__dirname,'screenshots');
(async()=>{
 const account=JSON.parse(await fs.readFile(path.join(root,'.cache/excel-live-account.json'),'utf8'));
 assert.match(account.email,/^qa-excel-[a-f0-9-]+@example\.test$/);assert.equal(account.workspace,'QA Upload Excel (Data Sintetis)');
 const apiContext=await request.newContext({baseURL:api,extraHTTPHeaders:{Origin:origin}});
 const login=await apiContext.post('/api/v1/auth/login',{data:{email:account.email,password:account.password}});assert.equal(login.status(),200);
 const me=await (await apiContext.get('/api/v1/auth/me')).json();assert.equal(me.user.email,account.email);
 let state=await (await apiContext.get('/api/v1/umkm/state')).json();assert.equal(state.workspace.name,account.workspace);
 if(state.state.entries.length){const clear=await apiContext.delete('/api/v1/umkm/data',{headers:{'X-CSRF-Token':me.csrf},data:{password:account.password,confirm:'HAPUS'}});assert.equal(clear.status(),200);}
 await apiContext.storageState({path:path.join(root,'.cache/excel-live-storage.json')});
 const browser=await chromium.launch({channel:process.env.BROWSER_CHANNEL||'chrome',headless:true});
 const context=await browser.newContext({storageState:path.join(root,'.cache/excel-live-storage.json'),serviceWorkers:'block',reducedMotion:'reduce',viewport:{width:1440,height:1000}});
 const page=await context.newPage();const errors=[],calls=[],checks=[];page.setDefaultTimeout(18000);page.on('pageerror',e=>errors.push(e.message));page.on('dialog',d=>d.accept());
 await context.addInitScript(()=>{window.__xlsxReads=0;for(const method of ['text','arrayBuffer']){const original=Blob.prototype[method];Blob.prototype[method]=function(...args){if(this instanceof File&&this.name.endsWith('.xlsx')){window.__xlsxReads++;throw Error('XLSX must never be decoded by the client')}return original.apply(this,args)}}});
 const types={'.html':'text/html','.js':'text/javascript','.css':'text/css','.svg':'image/svg+xml','.png':'image/png','.jpg':'image/jpeg','.woff2':'font/woff2','.csv':'text/csv','.webmanifest':'application/manifest+json'};
 const config=JSON.parse(await fs.readFile(path.join(root,'vercel.json'),'utf8'));
 await context.route(origin+'/**',async route=>{
  const url=new URL(route.request().url()),name=decodeURIComponent(url.pathname);let file;
  if(name==='/app')file=path.join(root,'frontend/index.html');
  else if(name.startsWith('/assets/'))file=path.join(root,'frontend',name);
  else if(name.startsWith('/pwa/'))file=path.join(root,name);
  else return route.continue();
  return route.fulfill({status:200,body:await fs.readFile(file),contentType:types[path.extname(file)]||'application/octet-stream',headers:Object.fromEntries(config.headers[0].headers.map(x=>[x.key,x.value]))});
 });
 page.on('response',async response=>{if(response.url().includes('/api/v1/umkm/imports')){let data;try{data=await response.json()}catch{}calls.push({path:new URL(response.url()).pathname,status:response.status(),count:data?.count,imported:data?.imported,auto_matched:data?.auto_matched,ai_matched:data?.ai_matched,ai_status:data?.ai_status,source:data?.source});}});
 try{
  await fs.mkdir(shotDir,{recursive:true});await page.goto(origin+'/app#imports');await page.locator('.cash-header').waitFor();
  await page.locator('#import-sale').setInputFiles(path.join(fixtureDir,'penjualan-20.xlsx'));await page.locator('#import-receipt').setInputFiles(path.join(fixtureDir,'mutasi-bank-21.xlsx'));
  await page.locator('#flex-import button[type=submit]').click();await page.locator('[data-action=flow-save]').waitFor();
  assert.match(await page.locator('#flow-preview').innerText(),/20 baris siap/);assert.match(await page.locator('#flow-preview').innerText(),/21 baris siap/);
  await page.evaluate(()=>document.activeElement?.blur());await page.screenshot({path:path.join(shotDir,'01-pratinjau-xlsx-desktop.png'),fullPage:true});checks.push('XLSX 20 sale + 21 receipt previews from live API');
  await page.locator('[data-action=flow-save]').click();await page.getByText(/21 baris tersimpan/).waitFor();await page.locator('.import-result .ai-status').last().waitFor();
  assert.equal(await page.locator('.import-result .ai-status').count(),2);await page.evaluate(()=>document.activeElement?.blur());await page.screenshot({path:path.join(shotDir,'02-hasil-impor-desktop.png'),fullPage:true});checks.push('canonical response.csv committed; actual AI status shown');
  await page.locator('nav a[href="#verifikasi"]').first().click();await page.getByRole('heading',{name:'Verifikasi',exact:true}).waitFor();await page.evaluate(()=>document.activeElement?.blur());await page.screenshot({path:path.join(shotDir,'03-verifikasi-desktop.png'),fullPage:false});checks.push('verification page uses live backend state');
  if(await page.locator('.automatic-results').count()){await page.locator('.automatic-results summary').click();await page.evaluate(()=>document.activeElement?.blur());await page.screenshot({path:path.join(shotDir,'04-beres-otomatis-desktop.png'),fullPage:true});checks.push('automatic backend allocations labeled separately with undo action');}
  await page.setViewportSize({width:375,height:900});await page.evaluate(()=>{location.hash='imports'});await page.locator('#import-sale').waitFor();await page.locator('#import-sale').setInputFiles(path.join(fixtureDir,'dua-sheet.xlsx'));await page.locator('#flex-import button[type=submit]').click();await page.locator('.upload-sheet-dialog').waitFor();
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),false);await page.evaluate(()=>document.activeElement?.blur());await page.screenshot({path:path.join(shotDir,'05-pilih-sheet-mobile.png')});
  await page.locator('#upload-sheet-choice').selectOption('Rekap');await page.getByRole('button',{name:'Periksa sheet',exact:true}).click();await page.locator('[data-action=flow-save]').waitFor();assert.match(await page.locator('#flow-preview').innerText(),/Sheet Rekap/);assert.match(await page.locator('#flow-preview').innerText(),/1 baris siap/);await page.evaluate(()=>document.activeElement?.blur());await page.screenshot({path:path.join(shotDir,'06-sheet-terpilih-mobile.png'),fullPage:true});checks.push('real 422 sheet selection on 375 px, same workbook resent with sheet=Rekap');
  await page.evaluate(()=>{location.hash='verifikasi'});await page.getByRole('heading',{name:'Verifikasi',exact:true}).waitFor();assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),false);await page.evaluate(()=>document.activeElement?.blur());await page.screenshot({path:path.join(shotDir,'07-verifikasi-mobile.png'),fullPage:false});
  await page.evaluate(()=>{location.hash='imports'});await page.locator('#import-sale').waitFor();await page.locator('#import-sale').setInputFiles(path.join(fixtureDir,'csv-lama.csv'));await page.locator('#flex-import button[type=submit]').click();await page.locator('[data-action=flow-save]').waitFor();await page.locator('[data-action=flow-save]').click();await page.locator('.import-preview[aria-label="Pratinjau Penjualan"] .success').filter({hasText:/^1 baris tersimpan/}).waitFor();checks.push('legacy 7-column CSV uploaded and committed through the real file endpoint');
  assert.equal(await page.evaluate(()=>window.__xlsxReads),0);assert.deepEqual(errors,[]);
  const final=await (await apiContext.get('/api/v1/umkm/state')).json();assert.equal(final.state.entries.length,42);
  const report={data:'isolated synthetic QA workspace',frontend:'branch source served at production origin using static-only Playwright overrides',api,sharedWorkspaceChanged:false,productionFrontendChanged:false,checks,calls,pageErrors:errors,clientXlsxReads:0,entries:final.state.entries.length};
  await fs.writeFile(path.join(root,'test-results/excel-live-e2e.json'),JSON.stringify(report,null,2));await fs.writeFile(path.join(__dirname,'provenance.json'),JSON.stringify(report,null,2));console.log(JSON.stringify({checks,entries:42,pageErrors:errors,clientXlsxReads:0}));
 }finally{await browser.close();await apiContext.dispose()}
})().catch(e=>{console.error(e);process.exit(1)});
