const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const assert=require('node:assert/strict'),fs=require('node:fs/promises'),path=require('node:path');
const base=process.env.GUIDE_URL||'http://127.0.0.1:3060';
const root=path.resolve(__dirname,'../../..');
const index=JSON.parse(require('node:fs').readFileSync(path.join(root,'dist/panduan/search-index.json'),'utf8'));
(async()=>{
 const browser=await chromium.launch({channel:'chrome',headless:true});const checks=[],errors=[],violations=[],failures=[];
 const context=await browser.newContext({serviceWorkers:'block',reducedMotion:'reduce'});
 await context.addInitScript(()=>{window.__csp=[];document.addEventListener('securitypolicyviolation',e=>window.__csp.push({directive:e.effectiveDirective,blocked:e.blockedURI}));});
 const p=await context.newPage();p.on('pageerror',e=>errors.push(e.message));p.on('response',r=>{if(r.status()>=400)failures.push({url:r.url(),status:r.status()})});
 try {
  for(const width of [375,768,1440])for(const item of index){
   await p.setViewportSize({width,height:width===375?812:1000});const response=await p.goto(base+item.url);assert.equal(response.status(),200,item.url);
   const csp=response.headers()['content-security-policy'];assert.match(csp,/script-src 'self'(?:;|$)/);assert.doesNotMatch(csp,/unsafe-inline|unsafe-eval|wasm-unsafe-eval/);
   await p.locator('h1').waitFor();assert.equal(await p.locator('h1').innerText(),item.title);
   assert.equal(await p.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),false,item.url+' overflow '+width);
   violations.push(...await p.evaluate(()=>window.__csp));checks.push({page:item.url,width,pass:true});
  }
  await p.setViewportSize({width:375,height:812});await p.goto(base+'/panduan/');
  await p.locator('starlight-menu-button button').click();assert.equal(await p.locator('#starlight__sidebar').isVisible(),true);await p.locator('starlight-menu-button button').click();
  for(const query of ['upload','rekonsiliasi','unduh','sandi','zzzzTidakAda']){
   await p.locator('.guide-search-open').click();await p.locator('#guide-query').fill(query);
   if(query.startsWith('zzzz'))await p.locator('[data-search-status]').filter({hasText:'Belum ada hasil'}).waitFor();
   else {await p.locator('[data-search-results] a').first().waitFor();assert.ok(await p.locator('[data-search-results] a').count()>0,query);}
   await p.keyboard.press('Escape');await p.waitForFunction(()=>document.activeElement===document.querySelector('.guide-search-open'));
  }
  await p.keyboard.press('Control+k');await p.locator('#guide-query').waitFor();await p.keyboard.press('Escape');
  const videoChecks=[];
  for(const slug of ['impor','verifikasi','ringkasan']){
   const requests=[];const listener=r=>{if(r.url().endsWith('.mp4'))requests.push(r.url())};p.on('request',listener);
   await p.goto(base+'/panduan/tugas/'+slug+'/');await p.waitForTimeout(100);assert.equal(requests.length,0,'video should not download before playback');
   await p.getByRole('link',{name:'Tonton video',exact:true}).click();const v=p.locator('video');
   assert.equal(await v.getAttribute('preload'),'none');assert.equal(await v.getAttribute('autoplay'),null);assert.equal(await v.locator('track[kind=captions][srclang=id]').count(),1);
   await v.evaluate(video=>{video.muted=true;video.textTracks[0].mode='showing';return video.play()});await p.waitForFunction(()=>document.querySelector('video').readyState>=2);
   const info=await v.evaluate(video=>({duration:video.duration,width:video.videoWidth,height:video.videoHeight,cues:video.textTracks[0]?.cues?.length||0}));
   assert.ok(info.duration>=45&&info.duration<=90);assert.equal(info.width,1280);assert.equal(info.height,720);assert.ok(info.cues>0);
   await p.locator('[data-video-speed]').selectOption('1.5');assert.equal(await v.evaluate(video=>video.playbackRate),1.5);await v.evaluate(video=>video.pause());
   await p.getByText('Baca transkrip video',{exact:true}).click();assert.ok(await p.locator('.tutorial-video details li').count()>=6);videoChecks.push({slug,...info});p.off('request',listener);
  }
  await p.goto(base+'/panduan/tugas/impor/');await p.route('**/panduan/video/*.mp4',r=>r.abort());await p.locator('video').evaluate(v=>{v.load();v.play().catch(()=>{})});await p.locator('.video-error').waitFor();assert.ok(await p.locator('#langkah').count());await p.unroute('**/panduan/video/*.mp4');
  await p.goto(base+'/panduan/');await p.route('**/panduan/search-index.json',r=>r.fulfill({status:503,body:'unavailable'}));await p.locator('.guide-search-open').click();await p.locator('[data-search-status]').filter({hasText:'Pencarian belum termuat'}).waitFor();await p.unroute('**/panduan/search-index.json');await p.keyboard.press('Escape');await p.locator('.guide-search-open').click();await p.locator('#guide-query').fill('CSV');await p.locator('[data-search-results] a').first().waitFor();await p.keyboard.press('Escape');
  assert.deepEqual(errors,[]);assert.deepEqual(violations,[]);assert.deepEqual(failures.filter(x=>![503].includes(x.status)),[]);
  const dir=path.join(root,'test-results');await fs.mkdir(dir,{recursive:true});await fs.writeFile(path.join(dir,'panduan.json'),JSON.stringify({base,views:checks.length,checks,videoChecks,cspViolations:violations,pageErrors:errors,search:'synonyms, empty, retry, keyboard passed',mobileMenu:'passed',videoErrorFallback:'passed'},null,2));
  for(const width of [1440,375]){await p.setViewportSize({width,height:width===375?812:1000});await p.goto(base+'/panduan/');await p.screenshot({path:path.join(dir,'panduan-'+width+'.png')});}
  console.log(JSON.stringify({views:checks.length,videos:videoChecks,cspViolations:violations,pageErrors:errors,mobileMenu:true,search:true,errorFallbacks:true}));
 }finally {await browser.close()}
})().catch(e=>{console.error(e);process.exit(1)});
