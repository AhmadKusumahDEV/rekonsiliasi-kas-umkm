const {fixtureRequest}=require('../../frontend/tests/browser/fixture-request.cjs');
const assert=require('node:assert/strict');
function setupFixture(page,{empty=false,authenticated=true}={}) {
 const entry=(id,kind,amount,extra={})=>({id:id.repeat(32),source_id:id,kind,amount,reference:'INV-102',name:'Rani',channel:'qris',date:'2026-11-03',canceled:false,...extra});
 const sales=[entry('a','sale',200000),entry('c','sale',50000,{reference:'HARIAN-03',name:'Penjualan harian',channel:'tunai',category:'cash',settled:true})];
 const receipts=[entry('b','receipt',190000)];
 const data={workspace:{id:'w',name:'Warung Rani (Simulasi)',version:1},state:{entries:empty?[]:[...sales,...receipts],imports:[],allocations:[]},events:[],users:[]};
 const user={id:'u',workspace_id:'w',name:'Rani (Simulasi)',email:'rani@example.test',role:'owner'};
 const flags={data,authenticated,calls:[],saved:[],allocated:false,previewItems:{}};
 page.addInitScript(()=>window.EventSource=class{constructor(){this.readyState=0}addEventListener(){}close(){}});
 page.route('**/api/v1/**',async route=>{
  const req=route.request(),url=new URL(req.url());
  assert(['127.0.0.1','localhost'].includes(url.hostname),'Documentation fixtures must stay local');
  const path=url.pathname,method=req.method(),body=await fixtureRequest(req);flags.calls.push({path,method});
  const send=(value,status=200)=>route.fulfill({status,contentType:'application/json',body:JSON.stringify(value)});
  if(path.endsWith('/config'))return send({kind:'umkm',brand:'EFA',limits:{max_import_rows:500,max_csv_bytes:1048576,max_entries:10000}});
  if(path.endsWith('/auth/me'))return send(flags.authenticated?{user,csrf:'documentation-fixture'}:{message:'Masuk dahulu'},flags.authenticated?200:401);
  if(path.endsWith('/auth/login')){flags.authenticated=true;return send({user,csrf:'documentation-fixture'})}
  if(path.endsWith('/umkm/state'))return send(data);
  if(path.endsWith('/umkm/audit'))return send(data.events);
  if(path.endsWith('/umkm/suggestions'))return send([]);
  if(path.endsWith('/umkm/exceptions')){
   if(flags.allocated||!data.state.entries.some(x=>x.kind==='receipt'))return send([]);
   return send([sales[0],receipts[0]].map((e,i)=>({entry:e,status:'unmatched',remaining:e.amount,candidate_count:0,candidate_ids:[],suggestion_available:false,reason:'Nominal berbeda. Periksa bukti potongan layanan.',...(i===0?{fee_hint:{amount:10000,label:'Biaya layanan',source:'bank',confidence:'tinggi',reason:'Saran Rp10.000 perlu dikonfirmasi dari bukti.'}}:{})})));
  }
  if(path.endsWith('/imports/files')){
   const items=body.kind==='sale'?(body.csv.includes('INV-102')?sales:[sales[1]]):receipts;flags.previewItems[body.kind]=items;
   return send({csv:'CANONICAL_'+body.kind,count:items.length,total:items.reduce((s,x)=>s+x.amount,0),skipped:0,warnings:body.kind==='receipt'?['Baris debit bukan uang masuk dan dilewati.']:[],preview:items,checksum:'simulation'});
  }
  if(path.endsWith('/umkm/imports')){
   flags.saved.push(body.kind);const items=flags.previewItems[body.kind]||(body.kind==='sale'?sales:receipts);
   for(const item of items)if(!data.state.entries.some(x=>x.id===item.id))data.state.entries.push(item);
   data.state.imports.push({id:String(data.state.imports.length+1).repeat(32),kind:body.kind,rows:items.length,canceled:false,created_at:'2026-11-03T08:00:00Z'});data.workspace.version++;
   return send({id:'d'.repeat(32),imported:items.length,auto_matched:0,ai_matched:0,skipped:0},201);
  }
  if(path.endsWith('/umkm/allocations')){
   flags.allocated=true;data.state.allocations.push({id:'f'.repeat(32),...body,reversed:false,created_at:'2026-11-03T08:00:00Z'});data.workspace.version++;
   data.events.push({id:1,actor:'Rani (Simulasi)',action:'Pasangkan pembayaran',detail:'Rp190.000 + biaya Rp10.000',created_at:'2026-11-03T08:00:00Z'});
   return send({id:'f'.repeat(32)},201);
  }
  if(path.endsWith('/umkm/export'))return route.fulfill({status:200,contentType:'text/csv',headers:{'Content-Disposition':'attachment; filename=umkm-rekap.csv'},body:'id,jenis,tanggal,referensi,nama,nominal,sisa,kanal,status,catatan\na,sale,2026-11-03,INV-102,Rani,200000,0,qris,matched,simulasi\n'});
  throw Error('Unexpected documentation fixture request: '+method+' '+path);
 });
 return flags;
}
module.exports={setupFixture};
