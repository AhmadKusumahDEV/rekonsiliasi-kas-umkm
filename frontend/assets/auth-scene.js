// shared presentation for sign-in and registration; authentication stays in app.js.
export function authScene(register = false) {
  return `<section class="auth-story" aria-label="Tentang Easy Financial Audit">
    <svg class="auth-shapes" width="0" height="0" aria-hidden="true"><defs><clipPath id="auth-panel-cut" clipPathUnits="objectBoundingBox"><path d="M.05 0H.78Q.86 0 .86 .065Q.86 .12 .94 .12H.95Q1 .12 1 .17V.95Q1 1 .95 1H.05Q0 1 0 .95V.05Q0 0 .05 0Z"/></clipPath></defs></svg>
    <div class="auth-editorial">
      <img class="auth-photo" src="/assets/${register ? "efa-finance-notes.jpg" : "efa-finance-desk.jpg"}" width="2400" height="3600" alt="" fetchpriority="high">
      <div class="auth-editorial-copy">
        <span class="auth-kicker">Untuk usaha yang terus berjalan</span>
        <h1>Jualannya jalan.<br>Catatannya jelas.</h1>
        <p>Cocokkan penjualan dengan uang yang masuk. Tahu mana yang beres, mana yang perlu dicek.</p>
      </div>
      <svg class="auth-spark" viewBox="0 0 240 240" fill="none" aria-hidden="true"><path d="m120 10 13 73 48-56-26 69 74-18-61 43 66 29-76-3 38 65-57-49-8 74-19-72-44 58 22-71-72 22 59-47-69-24 76-2-41-62 60 45Z" stroke="currentColor" stroke-width="1.5"/></svg>

    </div>
    <div class="auth-note">
      <span>Easy Financial Audit</span>
      <h2>Beda angka?<br>Temukan sebabnya.</h2>
      <p>Pembayaran sebagian, biaya layanan, atau dana yang belum cair. Periksa sebelum mengambil keputusan.</p>
      <img src="/assets/efa-mark.svg" width="50" height="50" alt="">
    </div>
      <a class="auth-photo-credit" href="${register ? "https://www.pexels.com/photo/a-person-working-on-the-documents-using-a-calculator-8297220/" : "https://www.pexels.com/photo/person-using-a-calculator-and-a-laptop-8296991/"}" target="_blank" rel="noopener noreferrer">Foto: Mikhail Nilov / Pexels ↗</a>
  </section>`;
}
