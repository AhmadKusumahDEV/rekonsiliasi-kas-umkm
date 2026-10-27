// Live is read-only except normal demo login. All scenario writes are localhost-only.
const fs = require("node:fs");
const path = require("node:path");
const crypto = require("node:crypto");
const {chromium} = require(process.env.PLAYWRIGHT_MODULE || "playwright");
const LIVE = "https://efa.madgo.my.id";
const LOCAL = "http://127.0.0.1:3044";
const root = __dirname;
const records = [], errors = [];
let p, source;
const categories = {
  "01-akses":"Akses akun", "02-ringkasan":"Ringkasan usaha",
  "03-transaksi":"Transaksi dan pencarian", "04-rekonsiliasi":"Pencocokan pembayaran",
  "05-impor":"Impor dan validasi CSV", "06-laporan":"Laporan dan jejak aktivitas",
  "07-pengaturan":"Pengaturan akun", "08-navigasi":"Navigasi dan halaman cadangan",
};
async function shot(cat,slug,title,text,steps,endpoint="") {
  await p.evaluate(()=>document.fonts.ready);
  await p.evaluate(()=>document.activeElement?.blur());
  await p.waitForTimeout(160);
  const viewport=p.viewportSize(),device=viewport.width<600?"mobile":"desktop";
  const file=path.join("screenshots",cat,device,slug+".png");
  fs.mkdirSync(path.dirname(path.join(root,file)),{recursive:true});
  if(await p.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1))throw Error("Overflow "+file);
  await p.screenshot({path:path.join(root,file),fullPage:true});
  records.push({category:cat,categoryLabel:categories[cat],slug,title,text,steps,endpoint,
    device,viewport,source,file:file.replaceAll("\\","/"),capturedAt:new Date().toISOString()});
  console.log("Captured "+source+" "+file);
}
async function route(hash){await p.evaluate(h=>{location.hash=h},hash);await p.waitForTimeout(100)}
async function close(){await p.locator('#modal [data-action="close"]').first().click();await p.waitForFunction(()=>!document.querySelector("#modal").open)}
async function localAPI(route,method="GET",body) {
  if(new URL(p.url()).origin!==LOCAL)throw Error("Writes are only allowed on the isolated localhost documentation server");
  return p.evaluate(async({route,method,body})=>{
    if(location.origin!=="http://127.0.0.1:3044")throw Error("Not local");
    let csrf="";
    if(method!=="GET"&&route!=="register")csrf=(await fetch("/api/me").then(r=>r.json())).csrf;
    const r=await fetch("/api/"+route,{method,headers:{"Content-Type":"application/json","X-CSRF-Token":csrf},
      body:body===undefined?undefined:JSON.stringify(body)});
    const data=await r.json();if(!r.ok)throw Error(data.error||data.message||r.status);return data;
  },{route,method,body});
}
const pages=[
 ["overview","02-ringkasan","01-ringkasan","Ringkasan usaha",
  "Melihat tagihan yang belum cocok, total penjualan, penerimaan dan kanal pembayaran. Angka memakai seluruh data aktif; penerimaan bukan laba.",
  ["Tentukan penjualan yang perlu ditelusuri.","Lanjutkan ke Cocokkan kas atau Impor CSV."],"GET /api/state"],
 ["entries","03-transaksi","01-transaksi","Daftar transaksi",
  "Penjualan dan penerimaan ditampilkan bersama nominal, sisa alokasi dan status. Pencarian serta filter membantu menelusuri catatan tertentu.",
  ["Cari nama atau referensi.","Gunakan filter dan pilihan 20/50/100 baris."],"GET /api/state"],
 ["reconcile","04-rekonsiliasi","01-cocokkan-kas","Usulan dan keputusan",
  "Usulan unik perlu diperiksa pemilik sebelum disimpan. Tabel keputusan menyimpan alokasi dan koreksi yang pernah dibuat.",
  ["Periksa pasangan yang diusulkan.","Pakai alokasi manual untuk sebagian, biaya atau pencairan gabungan."],"GET /api/suggestions"],
 ["imports","05-impor","01-batch-impor","Riwayat impor CSV",
  "Setiap file tersimpan sebagai batch. Pembatalan batch tidak menghapus jejak; batch beralokasi aktif perlu dikoreksi lebih dahulu.",
  ["Unduh template jika diperlukan.","Periksa CSV sebelum menyimpannya."],"GET /api/state"],
 ["history","06-laporan","01-riwayat","Jejak aktivitas",
  "Riwayat mencatat waktu, pelaksana, tindakan dan keterangan. Keputusan yang dibatalkan tetap dapat ditelusuri.",
  ["Cari tindakan yang diperlukan.","Bandingkan aktivitas dengan keputusan pencocokan."],"GET /api/state - events"],
 ["reports","06-laporan","02-laporan","Laporan dan ekspor",
  "Rekap memperlihatkan penjualan, penerimaan, sisa dan biaya yang dikonfirmasi. CSV dapat diunduh untuk pemeriksaan; ini bukan perhitungan laba.",
  ["Periksa ringkasan seluruh data.","Gunakan Unduh rekap CSV."],"GET /api/export"],
 ["settings","07-pengaturan","01-akun","Pengaturan akun",
  "Pemilik melihat identitas akun dan usaha yang sedang dibuka. Perubahan kata sandi tidak menghapus transaksi.",
  ["Pastikan ruang kerja yang dibuka benar.","Pilih Ganti kata sandi atau Keluar akun."],"GET /api/me dan GET /api/state"],
];
async function main(){
 const browser=await chromium.launch({channel:"chrome",headless:true});
 try{
  const context=await browser.newContext({viewport:{width:1440,height:960},deviceScaleFactor:1.5,reducedMotion:"reduce"});
  p=await context.newPage();p.on("pageerror",e=>errors.push(e.message));p.on("dialog",d=>d.accept());
  source="LIVE";
  await p.goto(LIVE,{waitUntil:"networkidle"});await p.locator('form[data-form=login]').waitFor();
  for(const width of [1440,390]){
   await p.setViewportSize({width,height:width<600?844:960});
   await shot("01-akses","01-masuk","Masuk ke akun","Login menggunakan email dan kata sandi, bukan memilih peran demo.",
    ["Gunakan email terdaftar.","Tekan Masuk untuk membuka usaha."],"POST /api/login");
   await p.getByRole("link",{name:"Daftar",exact:true}).click();
   await p.locator('form[data-form=register]').waitFor();
   await shot("01-akses","02-daftar","Mendaftarkan pemilik","Form meminta identitas pemilik dan nama usaha. Akun baru memiliki ruang kerja terpisah.",
    ["Isi nama, email, kata sandi dan nama usaha.","Dokumentasi tidak mengirim formulir daftar di server live."],"POST /api/register");
   await p.getByRole("link",{name:"Masuk",exact:true}).click();
  }
  await p.setViewportSize({width:1440,height:960});
  await p.locator('[name=email]').fill("pemilik@temukas.test");
  await p.locator('[name=password]').fill("KasDemo!2026");
  await p.locator('form[data-form=login] button[type=submit]').click();
  await p.locator(".cash-shell").waitFor();
  for(const width of [1440,390]){
   await p.setViewportSize({width,height:width<600?844:960});
   for(const [hash,cat,slug,title,text,steps,endpoint]of pages){
    await route(hash);await shot(cat,slug,title,text,steps,endpoint);
   }
   if(width===390){
    await p.locator('[data-action=mobile-menu]').click();
    await shot("08-navigasi","01-menu-hp","Menu navigasi HP","Menu membuka halaman yang sama dengan desktop, dalam ruang layar yang lebih kecil.",
     ["Pilih halaman yang diperlukan.","Tutup menu untuk kembali bekerja."],"Navigasi hash di browser");
    await p.locator('[data-action=close-menu]').click();
   }
  }
  // Use a separate local browser session and database for populated/error states.
  await context.close();
  const local=await browser.newContext({viewport:{width:1440,height:960},deviceScaleFactor:1.5,reducedMotion:"reduce"});
  p=await local.newPage();p.on("pageerror",e=>errors.push(e.message));p.on("dialog",d=>d.accept());source="LOCAL";
  await p.goto(LOCAL,{waitUntil:"networkidle"});await p.locator('form[data-form=login]').waitFor();
  await p.locator('[name=email]').fill("akun-tidak-terdaftar@example.test");
  await p.locator('[name=password]').fill("PasswordSalah!2026");
  await p.locator('form[data-form=login] button[type=submit]').click();
  await p.locator("[data-form-error]:visible").waitFor();
  await shot("01-akses","03-login-ditolak","Login belum cocok","Error berasal dari penolakan server lokal yang nyata. Pengguna tetap dapat memperbaiki isian.",
   ["Periksa email dan kata sandi.","Jangan mengulang percobaan tanpa memeriksa isian."],"401 dari POST /api/login");
  await localAPI("register","POST",{name:"Pemilik Demo",email:"ui-guide-"+Date.now()+"@example.test",
   password:"UIDoc!"+crypto.randomUUID()+"A",workspace:"Dapur Rasa (Simulasi)",role:"owner"});
  await p.reload({waitUntil:"networkidle"});await p.locator(".cash-shell").waitFor();
  await shot("02-ringkasan","02-usaha-baru","Ruang kerja tanpa transaksi","Keadaan kosong mengarahkan pemilik untuk mulai mengimpor catatan. Data kosong bukan kegagalan mengambil data.",
   ["Mulai dari Impor CSV.","Isi penjualan dan penerimaan secara terpisah."],"GET /api/state");
  const saleCSV=fs.readFileSync(path.join(root,"../../frontend/assets/contoh-penjualan.csv"),"utf8");
  const receiptCSV=fs.readFileSync(path.join(root,"../../frontend/assets/contoh-penerimaan.csv"),"utf8");
  await localAPI("imports","POST",{kind:"sale",csv:saleCSV});await localAPI("imports","POST",{kind:"receipt",csv:receiptCSV});
  await p.reload({waitUntil:"networkidle"});
  let data=await localAPI("state");
  const sale=ref=>data.state.entries.find(x=>x.kind==="sale"&&x.reference===ref);
  const receipt=ref=>data.state.entries.find(x=>x.kind==="receipt"&&x.reference===ref);
  for(const width of [1440,390]){
   await p.setViewportSize({width,height:width<600?844:960});await route("reconcile");
   await p.getByRole("button",{name:"Alokasi manual",exact:true}).click();
   await shot("04-rekonsiliasi","02-alokasi-manual","Form alokasi manual","Pilihan transaksi menampilkan sisa yang masih dapat dipakai. Server memeriksa saldo lagi ketika disimpan.",
    ["Pilih penjualan dan penerimaan yang benar.","Isi nominal lalu periksa sebelum menyimpan."],"POST /api/allocations");
   await p.locator('[name=sale_id]').selectOption(sale("INV-103").id);
   await p.locator('[name=receipt_id]').selectOption(receipt("INV-103").id);
   await p.locator('[name=amount]').fill("190000");await p.locator('[name=fee]').fill("10000");
   await p.locator('[name=reason]').fill("Biaya marketplace telah diperiksa.");
   await shot("04-rekonsiliasi","03-biaya-layanan","Memeriksa potongan biaya","Penerimaan Rp190.000 dan biaya Rp10.000 menjelaskan tagihan Rp200.000. Biaya tidak menambah uang yang diterima.",
    ["Jelaskan alasan biaya.","Periksa kedua transaksi sebelum menyimpan."],"amount + fee <= sisa penjualan");
   await close();await route("imports");await p.locator('[data-action=import]').first().click();
   await shot("05-impor","02-pilih-file","Memilih file CSV","Pilih jenis penjualan/penerimaan dan unduh template jika diperlukan. File dibaca sebagai teks oleh browser.",
    ["CSV UTF-8 maksimal 1 MB, 500 baris.","Periksa dahulu sebelum menyimpan."],"POST /api/imports/preview");
   await p.locator('[name=file]').setInputFiles({name:"contoh-penjualan.csv",mimeType:"text/csv",buffer:Buffer.from(saleCSV)});
   await p.getByRole("button",{name:"Periksa CSV",exact:true}).click();await p.locator('[data-action=commit-import]').waitFor();
   await shot("05-impor","03-pratinjau","Pratinjau CSV valid","Jumlah baris, total dan cuplikan muncul sebelum commit. Pratinjau tidak menambah transaksi.",
    ["Periksa nominal dan jenis.","Simpan hanya jika isinya benar."],"POST /api/imports/preview");
   await p.locator('[name=file]').setInputFiles({name:"contoh-salah.csv",mimeType:"text/csv",buffer:Buffer.from("kolom,salah\na,b")});
   await p.getByRole("button",{name:"Periksa CSV",exact:true}).click();await p.locator("[data-form-error]:visible").waitFor();
   await shot("05-impor","04-validasi-error","CSV ditolak oleh server","Header yang salah menghasilkan pesan koreksi. File tidak disimpan sebagian.",
    ["Gunakan tujuh kolom sesuai template.","Perbaiki file lalu periksa kembali."],"400 dari POST /api/imports/preview");
   await close();await route("settings");await p.locator('[data-action=password]').click();
   await shot("07-pengaturan","02-ganti-password","Form ganti kata sandi","Form meminta kata sandi saat ini dan kata sandi baru. Seluruh sesi berakhir setelah perubahan berhasil.",
    ["Isi kedua kata sandi.","Form ini tidak dikirim saat pengambilan screenshot."],"POST /api/password pada prototipe");
   await close();
  }
  await p.setViewportSize({width:1440,height:960});
  for(const [ref,amount,fee]of [["INV-101",100000,0],["INV-103",190000,10000],["INV-104",80000,0],["INV-105",120000,0]]){
   const rec=["INV-104","INV-105"].includes(ref)?data.state.entries.find(x=>x.kind==="receipt"&&!x.reference):receipt(ref);
   await localAPI("allocations","POST",{sale_id:sale(ref).id,receipt_id:rec.id,amount,fee,
    reason:fee?"Biaya marketplace telah diperiksa.":"Keputusan contoh sintetis.",key:crypto.randomUUID().replaceAll("-","")});
  }
  const decision=await localAPI("allocations","POST",{sale_id:sale("INV-102").id,receipt_id:receipt("INV-102").id,
   amount:100000,fee:0,reason:"Contoh koreksi.",key:crypto.randomUUID().replaceAll("-","")});
  await localAPI("allocations/"+decision.id+"/undo","POST",{});
  const canceled=await localAPI("imports","POST",{kind:"sale",csv:"id,tanggal,referensi,nama,nominal,kanal,catatan\nBATAL,2026-10-27,INV-BATAL,Contoh Batal,10000,tunai,Simulasi"});
  await localAPI("imports/"+canceled.id+"/cancel","POST",{});
  await p.reload({waitUntil:"networkidle"});
  for(const [hash,cat,slug,title,text,steps,endpoint]of pages.filter(x=>["overview","reconcile","imports","history","reports"].includes(x[0]))){
   await route(hash);await shot(cat,"10-skenario-"+slug,title+" - skenario contoh",
    text+" Ruang kerja lokal ini berisi pembayaran sebagian, biaya layanan, penerimaan gabungan dan koreksi.",
    steps,endpoint);
  }
  await route("entries");await p.locator("#list-status").selectOption("receipt");
  await shot("03-transaksi","02-filter","Filter penerimaan","Filter mengubah isi tabel tanpa menghapus data atau membatasi total usaha.",
   ["Pilih jenis transaksi.","Kembalikan ke Semua status untuk melihat semuanya."],"Filter lokal");
  await p.locator("#list-status").selectOption("");await p.locator("#list-search").fill("referensi-tidak-ada");
  await shot("03-transaksi","03-pencarian-kosong","Pencarian tanpa hasil","Pesan kosong memberi petunjuk mengganti kata kunci. Catatan usaha tetap tersimpan.",
   ["Ubah atau hapus pencarian.","Periksa filter yang masih aktif."],"Pencarian lokal");
  await p.locator("#list-search").fill("");await route("halaman-tidak-ada");
  await shot("08-navigasi","02-fallback","Halaman tidak tersedia","Hash yang tidak dikenal menampilkan tautan kembali. Ini fallback UI, bukan klaim HTTP 404 hosting.",
   ["Tekan Ringkasan.","Periksa tautan yang diketik manual."],"Fallback navigasi hash");
  const extra=["id,tanggal,referensi,nama,nominal,kanal,catatan"];
  for(let i=1;i<=25;i++)extra.push("PAGE-"+i+",2026-10-27,INV-PAGE-"+i+",Contoh "+i+",1000,tunai,Pagination");
  const batch=await localAPI("imports","POST",{kind:"sale",csv:extra.join("\n")});
  await p.goto(LOCAL+"/#entries",{waitUntil:"networkidle"});await p.locator('[data-action=next]').click();
  await shot("03-transaksi","04-pagination","Halaman kedua tabel","Batch lokal tambahan membuat halaman kedua tersedia. Ukuran halaman tidak membatasi ringkasan usaha.",
   ["Pilih 20/50/100 baris.","Gunakan Sebelumnya atau Berikutnya."],"Skenario pagination terpisah");
  await localAPI("imports/"+batch.id+"/cancel","POST",{});
  fs.writeFileSync(path.join(root,"manifest.json"),JSON.stringify({title:"Easy Financial Audit - Panduan UI",source:LIVE,
   capturedAt:new Date().toISOString(),timezone:"Asia/Bangkok",
   note:"LIVE screenshots are read-only; all scenario mutations use isolated localhost:3044 / rku_ui_guide.",
   runtimeErrors:errors,screenshots:records},null,2)+"\n");
  if(errors.length)throw Error(errors.join("\n"));
  console.log("Completed "+records.length+" screenshots; no page errors.");
 }finally{await browser.close()}
}
main().catch(e=>{console.error(e.message);process.exit(1)});
