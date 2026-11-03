const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'playwright');
const assert=require('node:assert/strict'),fs=require('node:fs/promises');
(async()=>{const b=await chromium.launch({channel:'chrome',headless:true});const p=await b.newPage({serviceWorkers:'block'});const errors=[];p.on('pageerror',e=>errors.push(e.message));const base=process.env.LANDING_URL||'http://127.0.0.1:3058';await fs.mkdir('test-results/landing-nov3',{recursive:true});let views=0;
for(const width of [375,768,1440])for(const route of ['/','/artikel/rekonsiliasi-penjualan-umkm','/artikel/saldo-rekening-bukan-laba','/artikel/rekonsiliasi-penjualan-umkm-2']){
 await p.setViewportSize({width,height:1000});const r=await p.goto(base+route);assert.equal(r.status(),200);
 await p.evaluate(async()=>{await document.fonts.ready;for(const img of document.images){if(img.loading==='lazy')img.loading='eager';await img.decode().catch(()=>{})}});
 for(const y of [0,800,1600,2400]){await p.evaluate(y=>scrollTo(0,y),y);await p.waitForTimeout(130)}
 assert.equal(await p.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,route+' '+width);
 assert.equal(await p.locator('img').evaluateAll(a=>a.filter(i=>i.getAttribute('src')&&(!i.complete||!i.naturalWidth)).length),0);
 assert.equal(await p.locator('.social-links a').count(),2);
 if(route!=='/'&&width===375){await p.locator('.menu-toggle').click();assert.equal(await p.locator('#mobile-nav').isVisible(),true);await p.keyboard.press('Escape');assert.equal(await p.locator('#mobile-nav').isVisible(),false);}
 if(route==='/'){
 assert.equal(await p.locator('#pause-slider').count(),0);assert.equal(await p.locator('.article-grid a.article-read').count(),3);
 for(const n of [1,2,0]){await p.locator('[data-slide="'+n+'"]').click();await p.waitForTimeout(550);assert.equal(await p.locator('[data-slide="'+n+'"]').getAttribute('aria-pressed'),'true');}
 for(const name of ['.capability-story','.closing-v2']){
  await p.locator(name).scrollIntoViewIfNeeded();await p.waitForTimeout(800);
  await p.locator(name).screenshot({path:'test-results/landing-nov3/'+name.slice(1)+'-'+width+'.png'});
 }
 await p.evaluate(()=>scrollTo(0,0));await p.waitForTimeout(800);await p.screenshot({path:'test-results/landing-nov3/home-'+width+'.png'});
 if(width===375){await p.locator('.menu-toggle').click();assert.equal(await p.locator('#mobile-nav').isVisible(),true);await p.keyboard.press('Escape');assert.equal(await p.locator('#mobile-nav').isVisible(),false);}
 }else {await p.evaluate(()=>scrollTo(0,0));await p.screenshot({path:'test-results/landing-nov3/article-'+width+'.png'});}
 views++;
}
await p.emulateMedia({reducedMotion:'reduce'});await p.goto(base+'/');await p.locator('[data-slide="2"]').click();await p.waitForTimeout(100);assert.equal(await p.locator('[data-slide="2"]').getAttribute('aria-pressed'),'true');assert.deepEqual(errors,[]);console.log(JSON.stringify({views,errors,images:'passed',articles:3}));await b.close();})().catch(e=>{console.error(e);process.exit(1)});
