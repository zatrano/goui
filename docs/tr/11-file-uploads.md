# 11. Dosya Yüklemeleri

GoUI'de ikili dosya verisi WebSocket üzerinden asla yolculuk etmez.
Yüklemeler, küçük bir `upload` paketi tarafından işlenen normal bir
multipart HTTP `POST`'tur; sadece ortaya çıkan *metadata* (bir ID, ad,
URL, boyut, içerik türü), normal bir olay aracılığıyla WS üzerinden bir
bileşene geri akar. Bu, WS protokolünü sadece-JSON tutar ve herhangi bir
bileşen koduna dokunmadan depolama arka ucunu değiştirmenize olanak
tanır.

Bu belge boyunca kullanılan modül yolu: `github.com/zatrano/goui`.

## 1. `upload.Storage`

```go
// upload/storage.go
type Storage interface {
	Save(originalName, contentType string, r io.Reader, size int64) (Meta, error)
	Open(id string) (io.ReadCloser, Meta, error)
	Delete(id string) error
}

type Meta struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	URL         string    `json:"url"`
	StoredAt    time.Time `json:"-"`
}
```

GoUI'de yüklenen dosyalarla ilgilenen her şey — HTTP route'ları,
`forms.DragDropUpload` bileşeni, istemci `upload.js` modülü — dosyalarla
yalnızca bu arayüz ve `Meta` aracılığıyla konuşur. Adapter'ınızın `Store`
seçeneğine (veya `upload.NewHandler` / `upload.Mount`'a) geçirdiğiniz
geçirilen uygulamayı değiştirin, diğer her katman etkilenmez.

## 2. `LocalStore` — yerleşik uygulama

```go
// upload/storage.go
type LocalStore struct {
	Dir      string
	BaseURL  string // örn. /goui/files
	MaxBytes int64

	mu    sync.RWMutex
	index map[string]Meta
}

func NewLocalStore(dir, baseURL string, maxBytes int64) (*LocalStore, error)
```

Sıfır değerler geçirdiğinizde `NewLocalStore` tarafından uygulanan
varsayılanlar:

| Alan | Boş/sıfır olduğunda varsayılan |
|---|---|
| `BaseURL` | `/goui/files` |
| `MaxBytes` | `8 << 20` (8 MiB) |

`Dir`'in varsayılanı yoktur — her zaman gerçek bir dizin sağlamanız
gerekir; `NewLocalStore`, önceden var olması gerekmediği için
`os.MkdirAll(dir, 0o755)`'i çağırır.

```go
store, err := upload.NewLocalStore("./data/uploads", "/goui/files", 8<<20)
if err != nil {
	log.Fatal(err)
}
```

### 2.1 `Save`

- Rastgele 16 baytlık bir hex ID (`newID`) üretir, orijinal dosya adını
  sanitize eder (`sanitizeName` — `filepath.Base` + literal bir `".."`
  değiştirme aracılığıyla dizin bileşenlerini ve `..` dizilerini
  soyar, dolayısıyla dosya adı aracılığıyla yol geçişi (path traversal)
  mümkün değildir), ve dosyayı `<ext>`'in *sanitize edilmiş* addan
  alındığı `<Dir>/<id><ext>`'e yazar.
- `MaxBytes`'ı **iki kez** zorlar: çağıran `size`'ı zaten önceden
  biliyorsa hızlı-yol reddi, ve kopyalarken sıkı bir dur
  (`io.LimitReader(r, MaxBytes+1)` — kopya `MaxBytes`'tan fazla
  okursa, kısmi dosya kaldırılır ve `"file too large"` döndürülür). Bu,
  çağıran `size` hakkında yalan söylese (veya önceden bilmese) bile
  `MaxBytes`'ın zorlanması anlamına gelir.
- Hiçbir `Content-Type` sağlanmadıysa `mime.TypeByExtension`'a, ve
  nihayetinde `application/octet-stream`'e geri döner.
- Ortaya çıkan `Meta`'yı (`URL: BaseURL + "/" + id` ile) bir bellek içi
  indekste (`sync.RWMutex` arkasında bir `map[string]Meta`) kaydeder.
  Bu indeks **kalıcı değildir** — process'i yeniden başlatmak, dosya
  hâlâ diskte olsa bile, önceden yüklenmiş dosyaları ID ile
  `Open`/`Delete` aracılığıyla sunma yeteneğini kaybeder. Yerel
  geliştirmenin ötesindeki her şey için buna göre planlayın (üretim
  şeklindeki alternatif için §5'e bakın).

### 2.2 `Open` / `Delete`

`Open(id)`, ID'yi bellek içi indekste arar, sonra diskte dosyayı bulmak
için `<Dir>/<id>.*`'i glob'lar (bir uzantısız yola geri dönerek).
`Delete(id)`, indeks girdisini kaldırır ve eşleşen dosyayı/dosyaları
glob'layıp kaldırır. `sync.RWMutex` sayesinde her ikisi de `Save` ile
eşzamanlı olarak çağrılmak için güvenlidir.

## 3. HTTP route'ları

Çekirdek modül framework-agnostic bir `net/http` handler sunar:

```go
// upload/handler.go
const (
    UploadPath  = "/goui/upload"
    FilesPrefix = "/goui/files"
)

func NewHandler(store Storage) *Handler   // POST upload + GET download
func Mount(mux *http.ServeMux, store Storage)
```

Yüklemeleri adapter'ınızın `Store` seçeneğiyle bağlayın (önerilen), veya
handler'ı doğrudan `net/http` üzerinde mount edin:

```go
store, _ := upload.NewLocalStore(filepath.Join(root, "data", "uploads"), "/goui/files", 8<<20)

// Fiber (WebSocket + upload tek çağrıda)
gouifiber.Register(app, gouifiber.Options{
    Server: ws.NewServer(hub, registry, tr),
    Store:  store,
})

// net/http ServeMux
gouistdlib.Register(mux, gouistdlib.Options{Server: server, Store: store})

// Veya yalnızca handler, herhangi bir http.Handler mux üzerinde:
upload.Mount(mux, store)
```

- **`POST /goui/upload`**, bir `file` alanı olan bir multipart form
  bekler (`c.FormFile("file")`). Başarıda `200` ve `Meta` JSON gövdesiyle
  yanıt verir (`{"id":..., "name":..., "contentType":..., "size":..., "url":...}`).
  Eksik bir dosyada veya bir `Storage.Save` hatasında (dosya çok büyük
  dahil) `{"error": "..."}` ile `400` yanıtı verir.
- **`GET /goui/files/:id`**, dosyayı saklanan `Content-Type`'ı ve bir
  `Content-Disposition: inline; filename="..."` başlığıyla geri
  akıtır, veya başarısızlıkta `404`/`500`. Mevcut handler'ın, yanıtı
  yazmadan önce dosyanın tamamını belleğe tampomladığını
  (`io.ReadAll`) unutmayın — bu paketin gönderdiği 8 MiB sınıfı
  varsayılanlar için sorun değil, ama `MaxBytes`'ı önemli ölçüde
  yükseltirseniz veya bu yol üzerinden büyük medya sunuyorsanız
  gözden geçirilmesi gereken bir şey.

Her iki route da, GoUI'ye özgü hiçbir auth'u olmayan düz HTTP
handler'larıdır — yüklemeler/indirmeler kısıtlanması gerekiyorsa, kendi
middleware'inizi `/goui/upload` ve `/goui/files`'ın önüne koyun.

## 4. İstemci tarafı akış, uçtan uca

`forms.DragDropUpload` (`forms/upload.go`), bir bırakma bölgesi artı
gizli bir "taşıyıcı" (carrier) düğme render eder:

```go
type DragDropUpload struct {
	core.BaseComponent
	CommonAttrs
	FieldValidation

	Files      []UploadedRef
	Accept     string
	Multiple   bool
	ShowThumbs bool
	UploadURL  string // varsayılan /goui/upload
	EventName  string
	OnChange   func(files []UploadedRef)
}
```

Tam gidiş-dönüş:

1. Render edilen markup, `data-goui-upload`, `data-upload-url`
   (varsayılan `/goui/upload`), `data-upload-event`, ve
   `data-accept`/`data-multiple` özniteliklerini taşır; bunlar istemci
   tarafından okunur.
2. `client/modules/upload.js` (`enhanceUpload`), herhangi bir
   `[data-goui-upload]` bölgesinde `dragover`/`drop`/`change`'i dinler,
   `accept`'e göre filtreler, ve her dosya için şunu çağırır:

   ```js
   export async function postFile(url, file) {
     const fd = new FormData();
     fd.append('file', file, file.name);
     const res = await fetch(url, { method: 'POST', body: fd });
     const data = await res.json();
     if (!res.ok) throw new Error(data.error || res.statusText);
     return data; // sunucudan Meta JSON'u
   }
   ```

3. Başarıda, `notifyUploaded(zone, meta)`, döndürülen `Meta`'yı
   bölgenin gizli `.goui-upload-carrier` düğmesindeki `data-goui-*`
   özniteliklerine tıkıştırır ve **sentetik olarak tıklar**:

   ```js
   carrier.setAttribute('data-goui-value', meta.id || '');
   carrier.setAttribute('data-goui-id', meta.id || '');
   carrier.setAttribute('data-goui-name', meta.name || '');
   carrier.setAttribute('data-goui-url', meta.url || '');
   carrier.setAttribute('data-goui-size', String(meta.size || 0));
   carrier.setAttribute('data-goui-content-type', meta.contentType || '');
   carrier.click();
   ```

4. Bu tıklama, `goui.js` tarafından delege edilen normal bir `g-click`
   olayıdır (`data-upload-event`, ki bu varsayılan olarak
   `<name>.uploaded`'dır); bu, aynı `data-goui-*` özniteliklerini olay
   payload'ına okur (`collectPayload`, `data-goui-id`, `-name`, `-url`,
   `-size`, `-content-type`, `-value` gibi diğerlerini tanır) ve onu
   normal bir WS olayı olarak gönderir.
5. `DragDropUpload.HandleEvent`, `"uploaded"` eylemini alır, payload'dan
   bir `UploadedRef` inşa eder, `d.Files`'ı ekler/değiştirir, ayarlıysa
   `OnChange`'i çağırır, ve `MarkDirty()`'yi çağırır — normal bir
   render, yeni dosyayı (ve `ShowThumbs`'sa ve o bir resimse küçük
   resmi) göstererek takip eder.

Diğer bir deyişle: **ikili yükleme, WS render döngüsünün tamamen
dışında, düz HTTP'dir; sadece ortaya çıkan metadata GoUI'ye yeniden
girer**, sıradan bir olay payload'ı olarak, türü bir checkbox
değiştirmeden veya bir metin değişikliğinden ayırt edilemez.
`forms.NewImageUpload(name, event)`, yaygın avatar/galeri durumu için
uygun bir preset'tir (`Accept: "image/*"`, `ShowThumbs: true`).

Kaldırma aynı şekli takip eder: listelenen her dosyadaki "×" düğmesi,
`"remove"` eylemine bağlı düz bir `g-click`'tir (`data-goui-value="<id>"`)
— dosyayı gerçekten silmek için herhangi bir HTTP çağrısı yapılmaz;
listeden kaldırmanın saklanan nesneyi de silmesi gerekiyorsa, bunu
`HandleEvent`'in `"remove"` dalında (`store.Delete(id)`'yi çağırarak)
kendiniz ekleyin.

## 5. Kendi `Storage`'ınızı yazma (S3/MinIO-şekilli iskelet)

Yukarıdaki her katman sadece `Storage` arayüzüne bağlı olduğundan, bir
bulut arka ucu eklemek üç metodu uygulama meselesidir. Aşağıda
**sadece-imza (signature-only) bir iskelet** vardır — kullandığınız
herhangi bir nesne deposu için SDK çağrılarını doldurun (AWS S3, MinIO,
Cloudflare R2, S3-uyumlu bir uç nokta aracılığıyla GCS vb.):

```go
package myupload

import (
	"context"
	"io"
	"time"

	"github.com/zatrano/goui/upload"
)

// S3Store, upload.Storage'ı bir S3-uyumlu bucket'a karşı uygular.
type S3Store struct {
	Client   S3API  // ihtiyacınız olan minimal client arayüzü (put/get/delete/head)
	Bucket   string
	BaseURL  string // örn. "https://cdn.example.com" veya presigned-URL tabanı
	MaxBytes int64
}

// S3API, S3Store'un ihtiyaç duyduğu minimal yüzeydir — bunu kendi SDK'nıza göre darlaştırın.
type S3API interface {
	PutObject(ctx context.Context, bucket, key string, body io.Reader, size int64, contentType string) error
	GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	HeadObject(ctx context.Context, bucket, key string) (contentType string, size int64, err error)
	DeleteObject(ctx context.Context, bucket, key string) error
}

func NewS3Store(client S3API, bucket, baseURL string, maxBytes int64) *S3Store {
	// LocalStore'un aynı varsayılan konvansiyonunu uygulayın: baseURL "" -> CDN/tabanınız,
	// maxBytes <= 0 -> 8<<20, vb.
	return &S3Store{Client: client, Bucket: bucket, BaseURL: baseURL, MaxBytes: maxBytes}
}

func (s *S3Store) Save(originalName, contentType string, r io.Reader, size int64) (upload.Meta, error) {
	// 1. size/MaxBytes koruması (size güvenilmezse LocalStore'un LimitReader hilesini yansıt)
	// 2. bir key üret, örn. zamana bölünmüş önek + rastgele id + sanitize edilmiş uzantı
	// 3. s.Client.PutObject(ctx, s.Bucket, key, r, size, contentType)
	// 4. {key -> orijinal ad / içerik türü / boyut}'u kalıcı bir yerde sakla —
	//    LocalStore'un bellek içi map'inin aksine, bu üretimde bir yeniden başlatmaya
	//    HAYATTA KALMALIDIR; paket seviyesi bir map değil, kendi uygulamanızın
	//    kendi veritabanı tablosunu kullanın.
	// 5. döndür upload.Meta{ID: key, Name: ..., ContentType: ..., Size: ..., URL: s.BaseURL + "/" + key, StoredAt: time.Now().UTC()}
	panic("not implemented")
}

func (s *S3Store) Open(id string) (io.ReadCloser, upload.Meta, error) {
	// 1. id için saklanan metadata'yı kalıcı deponuzdan arayın
	// 2. s.Client.GetObject(ctx, s.Bucket, id) (veya byte'ları uygulamanız üzerinden
	//    proxy'lemek yerine çağıranları presigned bir URL'ye yönlendirin — genellikle
	//    daha iyi üretim seçimi)
	panic("not implemented")
}

func (s *S3Store) Delete(id string) error {
	// 1. kalıcı metadata satırını kaldır
	// 2. s.Client.DeleteObject(ctx, s.Bucket, id)
	panic("not implemented")
}
```

Ardından tam olarak `LocalStore` gibi bağlayın:

```go
store := myupload.NewS3Store(s3Client, "my-bucket", "https://cdn.example.com", 8<<20)
gouifiber.Register(app, gouifiber.Options{Server: server, Store: store})
// GET /goui/files/:id, özel bir Storage'da presigned URL'ye 302 yönlendirebilir
```

`forms.DragDropUpload`, WS katmanı, veya istemci JS'sinin hiçbiri
değişmesi gerekmez — sadece `Meta`'yı (`id`, `name`, `contentType`,
`size`, `url`) ve iki HTTP route'unun JSON sözleşmesini görürler.

## 6. Pratik notlar

- **`MaxBytes`, sadece bir UI ipucu değil, sunucu tarafı sert bir
  sınırdır.** UX'i şekillendirmek için `DragDropUpload`'da
  `accept`/`Multiple`'ı ayarlayın, ama el yapımı bir istek herhangi bir
  istemci tarafı kontrolünü atlayabildiğinden, her zaman `MaxBytes`'ı
  da gerçekten zorlamak istediğiniz sınıra yapılandırın.
- **`/goui/upload` ve `/goui/files/:id`, varsayılan olarak kimlik
  doğrulamasızdır.** Yüklemeler/indirmeler herkese açık olması
  amaçlanmadıysa, kendi auth middleware'inizi bunların önüne uygulayın.
- **Bellek içi `LocalStore.index`, bir yeniden başlatmaya hayatta
  kalmaz.** Yerel geliştirme ve paketlenmiş örnekler için
  tasarlanmıştır. Kalıcı olan herhangi bir şey için, gerçek bir
  veritabanı + disk/nesne deposuyla desteklenen kendi `Storage`'ınızı
  yazın, veya en azından bellek içi indeksi, başlangıçta diskten
  (veya gerçek bir tablodan) yeniden inşa edilen biriyle değiştirin.
- Bir `DragDropUpload` listesinden bir dosyayı silmek, siz
  `HandleEvent`'in `"remove"` dalında kendiniz bir `store.Delete(id)`
  çağrısı eklemediğiniz sürece sadece-UI bir işlemdir — bkz. §4.
</contents>
