import test from 'node:test';
import assert from 'node:assert/strict';
import vm from 'node:vm';
import {readFile} from 'node:fs/promises';
const source=await readFile(new URL('../../landing/assets/analytics.js',import.meta.url),'utf8');
function load(hostname,path='/') {
 const scripts=[];
 const context={location:{hostname,pathname:path,href:`https://${hostname}${path}?email=secret@example.test#private`},document:{referrer:'https://example.test/ref?secret=yes#private',createElement:()=>({}),head:{append:script=>scripts.push(script)}},window:{},URL,Date};
 vm.runInNewContext(source,context);return {context,scripts};
}
test('analytics strips URL parameters and fragments and loads only on production public pages',()=>{
 const {context,scripts}=load('efa.madgo.my.id','/artikel/saldo-rekening-bukan-laba');
 assert.equal(scripts.length,1);assert.equal(scripts[0].src,'https://www.googletagmanager.com/gtag/js?id=G-8YV3VKNXHK');
 const config=context.window.dataLayer[1];assert.equal(config[0],'config');assert.equal(config[2].page_location,'https://efa.madgo.my.id/artikel/saldo-rekening-bukan-laba');assert.equal(config[2].page_referrer,'https://example.test/ref');assert.equal(config[2].allow_google_signals,false);
});
test('analytics does not instrument local previews or the authenticated app',()=>{
 assert.equal(load('localhost').scripts.length,0);assert.equal(load('efa.madgo.my.id','/app').scripts.length,0);
});
