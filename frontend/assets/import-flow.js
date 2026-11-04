import {validateImportSize,importLimits} from './mvp.js?v=handoff1';
export function createImportFlow({api,getState,e,money,refresh}) {
  let slots, busy=false, error='', generation=0;
  const reset = () => { generation++; slots={sale:{},receipt:{}};busy=false;error=''; };
  reset();
  const label = kind => kind==='sale'?'Penjualan':'Mutasi bank';
  function sync() {
    const box=document.querySelector('#flow-preview');if(box)box.innerHTML=previewHTML();
    document.querySelectorAll('#flex-import input,#flex-import button[type=submit]').forEach(node=>node.disabled=busy);
  }
  function previewHTML() {
    const active=Object.values(slots).filter(x=>x.file);
    return `${error?`<p class="error" role="alert">${e(error)}</p>`:''}${busy?'<p role="status">Memproses file… Jangan kirim ulang.</p>':''}` + Object.entries(slots).map(([kind,slot])=>{
      const p=slot.preview,r=slot.result;
      return `<section class="import-preview" aria-label="Pratinjau ${label(kind)}"><h3>${label(kind)}</h3><p>${e(slot.file?.name||'Belum ada file dipilih')}</p>${slot.error?`<p class="error" role="alert">${e(slot.error)}</p>`:''}${p?`<p>${p.count} baris siap · ${e(money(p.total))} · ${p.skipped||0} duplikat dilewati. ${r?'Sudah diproses.':'Belum disimpan.'}</p>${p.warnings?.length?'<div class="warning"><strong>Periksa peringatan ini</strong><ul>'+p.warnings.map(w=>'<li>'+e(w)+'</li>').join('')+'</ul></div>':''}<div class="table-scroll"><table><thead><tr><th>Tanggal</th><th>Referensi</th><th>Nama</th><th>Nominal</th></tr></thead><tbody>${p.preview.slice(0,5).map(x=>`<tr><td>${e(x.date)}</td><td>${e(x.reference||'—')}</td><td>${e(x.name)}</td><td>${e(money(x.amount))}</td></tr>`).join('')}</tbody></table></div>`:''}${r?`<p class="success" role="status">${r.imported||0} baris tersimpan · ${r.auto_matched||0} pasangan otomatis · ${r.ai_matched||0} pasangan AI · ${r.skipped||0} dilewati.</p>`:''}</section>`;
    }).join('') + (active.length && active.every(x=>x.preview&&!x.error) && active.some(x=>!x.result&&x.preview.count>0)?`<button type="button" class="primary" data-action="flow-save" ${busy?'disabled':''}>Simpan file yang sudah diperiksa</button><p class="muted">Penjualan disimpan lebih dulu. Jika satu file gagal, hasil file yang sudah tersimpan tetap ditampilkan; retry tidak mengulang file tersebut.</p>`:'') + (active.length&&active.every(x=>x.preview)&&active.every(x=>x.preview.count===0||x.result)?'<p>Tidak ada file lain yang perlu disimpan. Buka <a href="#verifikasi">Verifikasi</a> untuk melihat keputusan yang tersisa.</p>':'');
  }
  function html() {
    return `<section class="panel import-flow"><h2>Dua catatan, satu alur</h2><p>Isi salah satu atau kedua slot. Hanya CSV; simpan Excel sebagai CSV UTF-8 terlebih dahulu. File yang kurang lengkap tetap bisa disimpan; pencocokan memerlukan kedua sisi.</p><form id="flex-import" data-form="flex-import"><div class="error" data-form-error role="alert" tabindex="-1" hidden></div><div class="import-slots">${['sale','receipt'].map(kind=>`<label for="import-${kind}"><strong>${label(kind)}</strong><span>${kind==='sale'?'Tanggal, nama, nominal, kanal; referensi boleh kosong.':'CSV rekap penerimaan atau mutasi bank. Saldo bukan nominal transaksi.'}</span><input type="file" id="import-${kind}" data-import-kind="${kind}" accept=".csv,text/csv" ${busy?'disabled':''}></label>`).join('')}</div><p><a href="/assets/template-penjualan.csv" download>Unduh template penjualan</a> · <a href="/assets/contoh-mutasi-bank.csv" download>Contoh mutasi bank</a></p><button type="submit" class="primary" ${busy?'disabled':''}>Periksa file</button></form><div id="flow-preview" aria-live="polite">${previewHTML()}</div></section>`;
  }
  function setFile(kind,file) { if(busy)return;slots[kind]={file};error='';generation++;sync(); }
  async function preview() {
    if(busy)return; if(!Object.values(slots).some(x=>x.file))throw Error('Pilih minimal satu file CSV.');
    busy=true;error='';const token=generation,epoch=getState().epoch;sync();
    try {
      for(const kind of ['sale','receipt']){
        const slot=slots[kind];if(!slot.file||slot.result)continue;slot.error='';slot.preview=null;
        try {
          if(!/\.csv$/i.test(slot.file.name))throw Error('Gunakan CSV, bukan XLSX, PDF atau gambar.');
          const limits=validateImportSize(slot.file,getState().config);
          let csv;try{csv=new TextDecoder('utf-8',{fatal:true}).decode(await slot.file.arrayBuffer())}catch{throw Error('Simpan file sebagai CSV UTF-8.')}
          const p=await api('/api/v1/umkm/imports/flexible',{method:'POST',body:{kind,csv}});
          if(token!==generation||epoch!==getState().epoch)return;
          if(!Array.isArray(p.preview)||!Array.isArray(p.warnings)||typeof p.csv!=='string'||!Number.isSafeInteger(p.count)||p.count<0||!Number.isSafeInteger(p.total)||p.total<0||(p.count>0&&!p.csv.trim()))throw Error('Pratinjau server tidak sesuai. Coba periksa lagi.');
          if(p.count+(p.skipped||0)>limits.max_import_rows)throw Error('Baris CSV melebihi batas server.');
          slot.preview=p;
        }catch(err){if(token!==generation||epoch!==getState().epoch)return;slot.error=err.message;}
        sync();
      }
    }finally{if(token===generation){busy=false;sync()}}
  }
  async function commit() {
    if(busy)return;const active=Object.values(slots).filter(x=>x.file);
    if(!active.length||active.some(x=>!x.preview||x.error))throw Error('Periksa semua file yang dipilih sebelum menyimpan.');
    busy=true;error='';const token=generation,epoch=getState().epoch;sync();
    try{
      const count=active.filter(x=>!x.result).reduce((n,x)=>n+x.preview.count,0);
      if(getState().data.state.entries.length+count>importLimits(getState().config).max_entries)throw Error('Jumlah data melebihi batas ruang kerja.');
      for(const kind of ['sale','receipt']){
        const slot=slots[kind];if(!slot.preview||slot.result||slot.preview.count===0)continue;
        const result=await api('/api/v1/umkm/imports',{method:'POST',body:{kind,csv:slot.preview.csv}});
        if(token!==generation||epoch!==getState().epoch)return;slot.result=result;sync();
      }
    }catch(err){if(token===generation)error=err.message+' Periksa hasil di bawah sebelum mencoba lagi.';}
    finally{
      if(token===generation&&epoch===getState().epoch){busy=false;sync();await refresh(true).catch(()=>{});sync()}
    }
  }
  return {html,setFile,preview,commit,reset,get busy(){return busy},get dirty(){return Object.values(slots).some(x=>x.file&&!x.result)}};
}
