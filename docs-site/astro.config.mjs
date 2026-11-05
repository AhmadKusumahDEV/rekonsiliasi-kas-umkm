import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: 'https://efa.madgo.my.id',
  base: '/panduan',
  trailingSlash: 'always',
  integrations: [starlight({
    title: 'Panduan EFA',
    description: 'Langkah memakai EFA, dari catatan pertama sampai rekap kas.',
    defaultLocale: 'root',
    locales: { root: { label: 'Bahasa Indonesia', lang: 'id' } },
    logo: { src: '../frontend/assets/efa-mark.svg', replacesTitle: false },
    favicon: '/favicon.svg',
    customCss: ['./src/styles/efa.css'],
    components: { Header: './src/components/Header.astro', Footer: './src/components/Footer.astro' },
    sidebar: [
      { label: 'Mulai di sini', items: [
        { label: 'Cari panduan', slug: '' }, { label: 'Mulai cepat', slug: 'mulai-cepat' },
      ] },
      { label: 'Memakai EFA', items: [
        { label: 'Ringkasan & catat harian', slug: 'tugas/ringkasan' },
        { label: 'Impor CSV', slug: 'tugas/impor' },
        { label: 'Verifikasi', slug: 'tugas/verifikasi' },
        { label: 'Transaksi', slug: 'tugas/transaksi' },
        { label: 'Laporan & ekspor', slug: 'tugas/laporan' },
        { label: 'Pengaturan', slug: 'tugas/pengaturan' },
        { label: 'Reset kata sandi', slug: 'tugas/reset-password' },
        { label: 'Kosongkan data', slug: 'tugas/kosongkan-data' },
      ] },
      { label: 'Memahami catatan', items: [
        { label: 'Format CSV', slug: 'referensi/format-csv' },
        { label: 'Status transaksi', slug: 'referensi/status' },
        { label: 'Istilah sehari-hari', slug: 'referensi/istilah' },
        { label: 'Cara pencocokan bekerja', slug: 'penjelasan/pencocokan' },
        { label: 'Mengapa ada selisih?', slug: 'penjelasan/selisih' },
        { label: 'Pertanyaan umum', slug: 'faq' },
      ] },
    ],
    lastUpdated: true,
    pagination: true,
    // The parent build runs Pagefind directly, avoiding shell path issues on Windows.
    pagefind: false,
    social: [],
  })],
});
