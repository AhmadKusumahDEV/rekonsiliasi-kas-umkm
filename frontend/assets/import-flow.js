import {importLimits} from './mvp.js?v=excel1';
import {validateUpload,aiStatusMessage,uploadErrorMessage} from './import-status.js?v=excel1';

export function createImportFlow({api,getState,e,money,refresh}) {
  let slots, busy=false, error='', generation=0, cancelSheet;
  const reset = () => { generation++;cancelSheet?.();slots={sale:{},receipt:{}};busy=false;error=''; };
  reset();
  const label = kind => kind==='sale'?'Penjualan':'Mutasi bank';
  function sync() {
    const box=document.querySelector('#flow-preview');if(box)box.innerHTML=previewHTML();
    document.querySelectorAll('#flex-import input,#flex-import button[type=submit]').forEach(node=>node.disabled=busy);
  }
  function progressHTML(slot,kind) {
    if(!slot.phase)return '';
    const percent=slot.phase==='upload'&&slot.total>0?Math.min(100,Math.floor(slot.loaded/slot.total*100)):null;
    const text=slot.phase==='sheet'?'Menunggu pilihan sheet.':slot.phase==='processing'?'File terkirim. Membaca dan memeriksa data…':slot.phase==='saving'?'Menyimpan hasil yang sudah diperiksa…':percent===null?'Mengunggah file…':`Mengunggah file · ${percent}%`;
    return `<div class="upload-progress" role="status"><label for="upload-progress-${kind}">${text}</label><progress id="upload-progress-${kind}" max="100" ${percent===null?'':`value="${percent}"`}>${percent===null?'Memproses':percent+'%'}</progress></div>`;
  }
  function previewHTML() {
    const active=Object.values(slots).filter(x=>x.file);
    return `${error?`<p class="error" role="alert">${e(error)}</p>`:''}` + Object.entries(slots).map(([kind,slot])=>{
      const p=slot.preview,r=slot.result,source=p?.source;
      return `<section class="import-preview" aria-label="Pratinjau ${label(kind)}"><h3>${label(kind)}</h3><p class="upload-file">${slot.file?`<strong>${e(slot.file.name)}</strong><span>${e(slot.file.size.toLocaleString('id-ID'))} byte</span>`:'Belum ada file dipilih'}</p>${progressHTML(slot,kind)}${slot.error?`<p class="error" role="alert">${e(slot.error)}</p>`:''}${p?`${source?`<p class="upload-source">Sumber ${e(source.format?.toUpperCase()||'file')}${source.sheet?' · Sheet '+e(source.sheet):''}${source.header_row?' · Header baris '+e(source.header_row):''}</p>`:''}${!r&&(source?.sheets?.length>1)?`<button type="button" class="small" data-action="flow-sheet" data-id="${kind}" ${busy?'disabled':''}>Pilih sheet lain</button>`:''}<p>${p.count} baris siap · ${e(money(p.total))} · ${p.skipped||0} duplikat dilewati. ${r?'Sudah diproses.':'Belum disimpan.'}</p>${p.warnings?.length?'<div class="warning"><strong>Periksa peringatan ini</strong><ul>'+p.warnings.map(w=>'<li>'+e(w)+'</li>').join('')+'</ul></div>':''}<div class="table-scroll"><table><thead><tr><th>Tanggal</th><th>Referensi</th><th>Nama</th><th>Nominal</th></tr></thead><tbody>${p.preview.slice(0,5).map(x=>`<tr><td>${e(x.date)}</td><td>${e(x.reference||'—')}</td><td>${e(x.name)}</td><td>${e(money(x.amount))}</td></tr>`).join('')}</tbody></table></div>`:''}${r?`<div class="import-result"><p class="success" role="status">${r.imported||0} baris tersimpan · ${r.auto_matched||0} pasangan otomatis · ${r.ai_matched||0} pasangan AI · ${r.skipped||0} dilewati.</p><p class="ai-status" data-ai-status="${e(r.ai_status||'unknown')}">${e(aiStatusMessage(r))}</p>${r.ai_status==='failed'?'<a href="#verifikasi">Tinjau temuan di Verifikasi</a>':''}</div>`:''}</section>`;
    }).join('') + (active.length && active.every(x=>x.preview&&!x.error) && active.some(x=>!x.result&&x.preview.count>0)?`<button type="button" class="primary" data-action="flow-save" ${busy?'disabled':''}>Simpan file yang sudah diperiksa</button><p class="muted">Penjualan disimpan lebih dulu. Jika satu file gagal, hasil file yang sudah tersimpan tetap ditampilkan; coba lagi tidak mengulang file tersebut.</p>`:'') + (active.length&&active.every(x=>x.preview)&&active.every(x=>x.preview.count===0||x.result)?'<p>Tidak ada file lain yang perlu disimpan. Buka <a href="#verifikasi">Verifikasi</a> untuk melihat keputusan yang tersisa.</p>':'');
  }
  function html() {
    return `<section class="panel import-flow"><h2>Dua catatan, satu alur</h2><p>Pilih CSV atau Excel (.xlsx), maksimal 10 MB per file. Isi salah satu atau kedua slot. Jika workbook berisi beberapa sheet, kamu akan diminta memilih satu; file belum disimpan saat diperiksa.</p><form id="flex-import" data-form="flex-import"><div class="error" data-form-error role="alert" tabindex="-1" hidden></div><div class="import-slots">${['sale','receipt'].map(kind=>`<label for="import-${kind}"><strong>${label(kind)}</strong><span>${kind==='sale'?'Tanggal, nama, nominal, kanal; referensi boleh kosong.':'Rekap penerimaan atau mutasi bank. Saldo bukan nominal transaksi.'}</span><input type="file" id="import-${kind}" data-import-kind="${kind}" accept=".csv,.xlsx,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" ${busy?'disabled':''}></label>`).join('')}</div><p><a href="/assets/template-penjualan.csv" download>Unduh template penjualan</a> · <a href="/assets/contoh-mutasi-bank.csv" download>Contoh mutasi bank</a></p><p class="muted">Hasil konversi tetap mengikuti batas server: maksimal 1 MB CSV dan 500 baris. XLS lama, PDF, dan file Excel terenkripsi belum didukung.</p><button type="submit" class="primary" ${busy?'disabled':''}>Periksa file</button></form><div id="flow-preview" aria-live="polite">${previewHTML()}</div></section>`;
  }
  function setFile(kind,file) { if(busy||!slots[kind])return;slots[kind]={file};error='';generation++;sync(); }
  function pickSheet(kind,slot,sheets) {
    return new Promise(resolve=>{
      const dialog=document.createElement('dialog');dialog.className='upload-sheet-dialog';dialog.setAttribute('aria-labelledby','upload-sheet-title');
      dialog.innerHTML=`<form method="dialog"><h2 id="upload-sheet-title">Pilih sheet ${label(kind)}</h2><p>${e(slot.file.name)} memiliki beberapa sheet. Pilih satu yang ingin diperiksa. File belum disimpan.</p><label for="upload-sheet-choice">Sheet<select id="upload-sheet-choice" name="sheet" required><option value="" disabled ${slot.sheet?'':'selected'}>Pilih satu sheet</option>${sheets.map(name=>`<option value="${e(name)}" ${slot.sheet===name?'selected':''}>${e(name)}</option>`).join('')}</select></label><div class="form-actions"><button type="submit" value="cancel" formnovalidate>Batal</button><button type="submit" class="primary" value="select">Periksa sheet</button></div></form>`;
      dialog.querySelector('form').addEventListener('submit',event=>event.stopPropagation());
      const finish=()=>{const selected=dialog.returnValue==='select'?dialog.querySelector('select').value:null;dialog.remove();cancelSheet=null;resolve(selected&&sheets.includes(selected)?selected:null);};
      dialog.addEventListener('close',finish,{once:true});cancelSheet=()=>dialog.close('cancel');document.body.append(dialog);dialog.showModal();
    });
  }
  async function uploadSlot(kind,slot,token,epoch) {
    validateUpload(slot.file);const limits=importLimits(getState().config);
    for(let attempt=0;attempt<2;attempt++){
      const form=new FormData();form.append('file',slot.file);form.append('kind',kind);if(slot.sheet)form.append('sheet',slot.sheet);
      slot.phase='upload';slot.loaded=0;slot.total=0;sync();
      try{
        const p=await api('/api/v1/umkm/imports/files',{method:'POST',body:form,onUploadProgress:progress=>{
          if(token!==generation||epoch!==getState().epoch)return;
          slot.phase=progress.phase;slot.loaded=progress.loaded||0;slot.total=progress.total||0;sync();
        }});
        if(token!==generation||epoch!==getState().epoch)return;
        if(!Array.isArray(p.preview)||!Array.isArray(p.warnings)||typeof p.csv!=='string'||!Number.isSafeInteger(p.count)||p.count<0||!Number.isSafeInteger(p.total)||p.total<0||(p.count>0&&!p.csv.trim()))throw Error('Pratinjau server tidak sesuai. Periksa lagi; belum ada data disimpan.');
        if(p.preview.some(row=>!row||typeof row!=='object')||p.warnings.some(item=>typeof item!=='string')||(p.source&&(typeof p.source.format!=='string'||(p.source.sheets&&(!Array.isArray(p.source.sheets)||p.source.sheets.length>20||p.source.sheets.some(name=>typeof name!=='string'||!name))))))throw Error('Metadata pratinjau server belum lengkap. Periksa lagi; belum ada data disimpan.');
        if(p.count+(p.skipped||0)>limits.max_import_rows||new TextEncoder().encode(p.csv).length>limits.max_csv_bytes)throw Error('Hasil konversi melebihi batas server. Kurangi baris atau pilih sheet yang lebih kecil.');
        slot.preview=p;return;
      }catch(err){
        if(token!==generation||epoch!==getState().epoch)return;
        const sheets=err.details?.sheets;
        if(err.status!==422||attempt>0||!Array.isArray(sheets)||!sheets.length||sheets.length>20||sheets.some(x=>typeof x!=='string'||!x))throw err;
        slot.phase='sheet';sync();const choice=await pickSheet(kind,slot,[...new Set(sheets)]);
        if(token!==generation||epoch!==getState().epoch)return;
        if(!choice)throw Error('Pemilihan sheet dibatalkan. Tekan Periksa file saat kamu siap melanjutkan.');slot.sheet=choice;
      }
    }
  }
  async function preview(onlyKind) {
    if(busy)return;if(!Object.values(slots).some(x=>x.file))throw Error('Pilih minimal satu file CSV atau XLSX.');
    busy=true;error='';const token=generation,epoch=getState().epoch;sync();
    try {
      for(const kind of ['sale','receipt']){
        const slot=slots[kind];if(!slot.file||slot.result||(onlyKind&&onlyKind!==kind))continue;slot.error='';slot.preview=null;
        try { await uploadSlot(kind,slot,token,epoch); }
        catch(err){if(token!==generation||epoch!==getState().epoch)return;slot.error=uploadErrorMessage(err);}
        if(token!==generation||epoch!==getState().epoch)return;slot.phase='';sync();
      }
    }finally{if(token===generation){busy=false;sync()}}
  }
  async function changeSheet(kind) {
    const slot=slots[kind];if(busy||!slot||slot.result||!Array.isArray(slot.preview?.source?.sheets))return;
    busy=true;sync();const token=generation,epoch=getState().epoch;
    try{const choice=await pickSheet(kind,slot,slot.preview.source.sheets);if(token!==generation||epoch!==getState().epoch)return;if(choice){slot.sheet=choice;slot.preview=null;slot.error='';}}
    finally{if(token===generation){busy=false;sync()}}
    if(token===generation&&!slot.preview)await preview(kind);
  }
  async function commit() {
    if(busy)return;const active=Object.values(slots).filter(x=>x.file);
    if(!active.length||active.some(x=>!x.preview||x.error))throw Error('Periksa semua file yang dipilih sebelum menyimpan.');
    busy=true;error='';const token=generation,epoch=getState().epoch;sync();
    try{
      const count=active.filter(x=>!x.result).reduce((n,x)=>n+x.preview.count,0);
      if(getState().data.state.entries.length+count>importLimits(getState().config).max_entries)throw Error('Jumlah data melebihi batas ruang kerja.');
      for(const kind of ['sale','receipt']){
        const slot=slots[kind];if(!slot.preview||slot.result||slot.preview.count===0)continue;slot.phase='saving';sync();
        const result=await api('/api/v1/umkm/imports',{method:'POST',body:{kind,csv:slot.preview.csv}});
        if(token!==generation||epoch!==getState().epoch)return;slot.result=result;slot.phase='';sync();
      }
    }catch(err){if(token===generation){error=uploadErrorMessage(err)+' Periksa hasil di bawah sebelum mencoba lagi.';if(err.status===409)for(const slot of Object.values(slots))if(!slot.result)slot.preview=null;}}
    finally{
      if(token===generation&&epoch===getState().epoch){busy=false;for(const slot of Object.values(slots))slot.phase='';sync();await refresh(true).catch(()=>{});sync()}
    }
  }
  return {html,setFile,preview,changeSheet,commit,reset,get busy(){return busy},get dirty(){return Object.values(slots).some(x=>x.file&&!x.result)}};
}
