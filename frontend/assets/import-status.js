export const MAX_UPLOAD_BYTES = 10 * 1024 * 1024;

export function validateUpload(file) {
  if(!file?.size) throw new Error('File kosong. Pilih CSV atau XLSX yang berisi data.');
  if(!/\.(csv|xlsx)$/i.test(file.name)) throw new Error('Gunakan CSV atau XLSX. File XLS lama, PDF, dan gambar belum didukung.');
  if(file.size>MAX_UPLOAD_BYTES) throw new Error('File melebihi 10 MB. Pecah workbook atau kurangi isinya, lalu unggah kembali.');
}

export function aiStatusMessage(result) {
  const count=Number.isSafeInteger(result?.ai_matched)&&result.ai_matched>=0?result.ai_matched:0;
  return ({
    ok:`${count} pasangan dicek AI dan disimpan otomatis`,
    no_candidates:'Tidak ada pasangan ambigu untuk AI',
    disabled:'Pemeriksaan AI tidak aktif; pencocokan memakai aturan',
    skipped:'Pemeriksaan AI dilewati; pencocokan memakai aturan',
    failed:'Pemeriksaan AI gagal. Hasil aturan tetap tersimpan; tinjau temuan manual',
  })[result?.ai_status]||'Status pemeriksaan AI belum tersedia dari server. Hasil impor tetap ditampilkan.';
}

export function uploadErrorMessage(error) {
  const next=({
    400:'Periksa format, header, dan baris yang disebutkan; simpan ulang file jika rusak atau dilindungi sandi.',
    413:'File maksimal 10 MB. Hasil konversi tetap maksimal 1 MB dan 500 baris; pecah data atau pilih sheet yang lebih kecil.',
    415:'Simpan sebagai CSV atau XLSX; jangan hanya mengganti ekstensi file.',
    422:'Pilih satu sheet berisi data, lalu periksa lagi. Jika daftar sheet tidak muncul, buka ulang workbook dan simpan sebagai XLSX.',
    409:'Data berubah atau berbenturan. Muat data terbaru dan periksa ulang file sebelum menyimpan lagi.',
    429:'Tunggu sebentar sebelum mencoba lagi. Jangan kirim berulang kali.',
    503:'Sesi atau database backend belum siap. Coba beberapa saat lagi; jangan menganggap impor sudah tersimpan.',
  })[error?.status];
  return [error?.message||'File belum dapat diproses.',next].filter(Boolean).join(' ');
}

// The current backend stores automation provenance in these fixed reason prefixes.
export function allocationOrigin(allocation) {
  const reason=allocation?.reason||'';
  if(/^Disarankan AI \(skor aturan [\d.]+\):/.test(reason)) return 'ai';
  if(/^Cocok otomatis \(skor [\d.]+\):/.test(reason)) return 'rules';
  return null;
}
export function readableAllocationReason(reason='') {
  return reason.replace(/^Disarankan AI \(skor aturan [\d.]+\):\s*/, 'Dicek AI: ')
    .replace(/^Cocok otomatis \(skor [\d.]+\):\s*/, 'Cocok otomatis: ');
}
