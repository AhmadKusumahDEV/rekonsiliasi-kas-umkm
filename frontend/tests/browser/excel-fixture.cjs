const {fixtureRequest}=require('./fixture-request.cjs');
function excelFixture(page) {
 const entry=(id,kind,amount=100000)=>({id:id.repeat(32),source_id:id,kind,amount,reference:'QA-ONE',name:'Pelanggan Simulasi',channel:'transfer',date:'2026-11-05',canceled:false});
 const sale=entry('a','sale'),receipt=entry('b','receipt');
 const data={workspace:{id:'w',name:'Uji Excel (Simulasi)',version:1},state:{entries:[sale,receipt],imports:[],allocations:[]},events:[],users:[]};
 const flags={data,calls:[],uploads:[],commits:[],errorStatus:0,commitStatus:0,ai_status:'ok',hold:false,sheets:['Penjualan','Rekap'],loggedIn:true,malformed:false,oversizedCanonical:false,badSource:false};
 page.addInitScript(()=>{window.EventSource=class{constructor(){this.readyState=0}addEventListener(){}close(){}};window.xlsxReads=0;for(const method of ['text','arrayBuffer']){const old=Blob.prototype[method];Blob.prototype[method]=function(...args){if(this instanceof File&&/\.xlsx$/i.test(this.name)){window.xlsxReads++;throw Error('Client XLSX read')}return old.apply(this,args)}}});
 page.route('**/api/v1/**',async route=>{
  const req=route.request(),url=new URL(req.url()),body=await fixtureRequest(req);flags.calls.push({path:url.pathname,method:req.method()});
  const send=(data,status=200)=>route.fulfill({status,contentType:'application/json',body:JSON.stringify(data)});
  if(url.pathname.endsWith('/config'))return send({kind:'umkm',brand:'EFA',limits:{max_import_rows:500,max_csv_bytes:1048576,max_entries:10000}});
  if(url.pathname.endsWith('/auth/me'))return flags.loggedIn?send({user:{id:'u',workspace_id:'w',name:'Pemilik Simulasi',email:'synthetic@example.test',role:'owner'},csrf:'fixture-csrf'}):send({message:'Sesi berakhir'},401);
  if(url.pathname.endsWith('/umkm/state'))return send(data);
  if(url.pathname.endsWith('/umkm/exceptions'))return send(data.state.allocations.length?[]:[sale,receipt].map(x=>({entry:x,status:'unmatched',remaining:x.amount,candidate_count:1,candidate_ids:[x.kind==='sale'?receipt.id:sale.id],suggestion_available:true,reason:'Periksa bukti.'})));
  if(url.pathname.endsWith('/umkm/suggestions'))return send(data.state.allocations.length?[]:[{sale,receipt,reason:'Pasangan unik dari server'}]);
  if(url.pathname.endsWith('/umkm/audit'))return send([]);
  if(url.pathname.endsWith('/imports/files')){
   flags.uploads.push({...body,csrf:req.headers()['x-csrf-token']});
   if(flags.hold)await new Promise(resolve=>flags.release=resolve);
   if(flags.errorStatus===401)flags.loggedIn=false;
   if(flags.errorStatus)return send({message:'File belum dapat diproses pada baris 8.'},flags.errorStatus);
   if(body.file==='dua-sheet.xlsx'&&!body.sheet)return send({message:'Pilih sheet',details:{sheets:flags.sheets}},422);
   return send(flags.malformed?{count:20,csv:null}:{csv:flags.oversizedCanonical?'x'.repeat(1048577):'CANONICAL_'+body.kind+'_'+(body.sheet||'default'),count:body.kind==='sale'?20:21,total:2000000,skipped:0,warnings:['Baris debit bukan uang masuk.'],preview:[sale],source:{format:flags.badSource?42:body.file.endsWith('.xlsx')?'xlsx':'csv',sheet:body.sheet||'Penjualan',header_row:8,sheets:body.file==='dua-sheet.xlsx'?flags.sheets:['Penjualan']}});
  }
  if(url.pathname.endsWith('/umkm/imports')){
   flags.commits.push(body);if(flags.commitStatus)return send({message:'Data berubah'},flags.commitStatus);
   data.state.allocations=[{id:'f'.repeat(32),sale_id:sale.id,receipt_id:receipt.id,amount:100000,fee:0,reason:flags.ai_status==='ok'?'Disarankan AI (skor aturan 0.90): bukti sesuai':'Cocok otomatis (skor 1.00): referensi sama',reversed:false}];data.workspace.version++;
   return send({id:'e'.repeat(32),imported:body.kind==='sale'?20:21,auto_matched:1,ai_matched:flags.ai_status==='ok'?2:0,ai_status:flags.ai_status,skipped:0},201);
  }
  if(url.pathname.endsWith('/reverse')){data.state.allocations[0].reversed=true;return send({ok:true})}
  return send({message:'Unexpected fixture path '+url.pathname},404);
 });
 return flags;
}
module.exports={excelFixture};
