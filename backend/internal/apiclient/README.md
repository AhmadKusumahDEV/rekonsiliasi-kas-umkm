# internal/apiclient

Satu folder per layanan eksternal, misalnya `internal/apiclient/payment/`.
Client menyimpan `*http.Client`, konfigurasi yang dibutuhkan, dan model request/
response. Client tidak boleh mengakses database, dan service tetap menjadi tempat
orkestrasi aturan bisnis.
