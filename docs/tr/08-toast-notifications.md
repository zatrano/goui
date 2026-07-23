# 08. Toast Bildirimleri

GoUI, önceki bölümlerde açıklanan WebSocket taşımasının (transport)
üzerine doğrudan inşa edilmiş küçük bir push bildirimi ("toast") sistemi
sunar. Ayrı bir HTTP uç noktası veya polling söz konusu değildir — bir
toast, bir bileşenin render için zaten kullandığı aynı bağlantı üzerindeki
sadece başka bir frame türüdür.

Bu belge boyunca kullanılan modül yolu: `github.com/zatrano/goui`.

## 1. Hareketli parçalar

| Parça | Konum | Rol |
|---|---|---|
| `core.BaseComponent.Toast` / `ToastT` | `core/component.go` | *Mevcut* oturum için bir toast ateşlemek için bileşen tarafı API. |
| `ws.PushMessage` | `ws/frame.go` | Tel payload'ı: `{Kind, Text}`. |
| `ws.FrameTypePush` | `ws/frame.go` | Bir `PushMessage` taşıyan frame türü (`"push"`). |
| `Session.EnqueuePush` | `ws/session.go` | Bir push frame'ini oturumun giden kanalına koyar. |
| `Hub.Push` | `ws/hub.go` | Bir toast'ı *bir* oturuma, oturum ID'sine göre gönderir. |
| `Hub.Broadcast` | `ws/hub.go` | Bir toast'ı kayıtlı *her* oturuma gönderir. |
| `client/modules/toast.js` | istemci çalışma zamanı | Toast DOM'unu render eder ve otomatik kapatmayı yönetir. |

### 1.1 `core.BaseComponent`

Her bileşen `core.BaseComponent`'i gömer, ki bu, bir bileşen mount veya
activate edildiğinde oturumun otomatik olarak bağladığı dahili bir
`pusher` callback'i tutar (bkz. `ws/session.go`'daki
`Session.injectPusher`). Bunu kendiniz asla ayarlamazsınız.

```go
// core/component.go (alıntı)

// Toast, mevcut oturuma bir push bildirimi gönderir (pusher yoksa no-op).
func (b *BaseComponent) Toast(kind, text string) {
	if b.pusher != nil {
		b.pusher(kind, text)
	}
}

// ToastT, key'i çevirir sonra mevcut oturum için bir toast gönderir.
func (b *BaseComponent) ToastT(kind, key string, args ...any) {
	b.Toast(kind, b.T(key, args...))
}
```

`Toast`, `pusher` `nil` olduğunda no-op olduğundan, her zaman çağırmak
güvenlidir — bir `Session` olmadan doğrudan bir bileşen inşa eden birim
testlerinden, veya canlı bir bağlantıya bağlanmadan önce bir bileşenin
`Mount`/`HandleEvent`'inden dahil.

- **`Toast(kind, text string)`** — literal bir string gönderir.
- **`ToastT(kind, key string, args ...any)`** — `key`'i bileşenin enjekte
  edilmiş `i18n.Translator`'ı aracılığıyla çözer (`T()` ile aynı
  mekanizma), sonra çevrilmiş string'i gönderir. Uygulama kodunda
  `ToastT`'yi tercih edin, böylece toast metni, diğer her kullanıcıya
  yönelik string'in yanında locale dosyalarınızda yaşar.

### 1.2 Türler (Kinds)

`Kind` alanı serbest biçimli bir string'tir, ama istemci ve varsayılan
stil sayfası sadece dört değere özel işlem verir:

```
success | error | warning | info
```

Diğer her şey (boş bir string dahil) istemci tarafından `info`'ya
normalize edilir (`client/modules/toast.js`'deki `normalizeKind`).
Sunucu `Kind`'i doğrulamaz — eşleşen CSS de göndermediğiniz sürece
yukarıdaki dört türden birini seçin.

### 1.3 Zamanlamalar

Toast'lar istemcide otomatik olarak kapanır. Kullanıcıların onları okumak
için daha fazla zamanı olsun diye hatalar için varsayılan ömür daha
uzundur:

```js
// client/modules/toast.js
const DEFAULT_MS = 5000; // success / warning / info
const ERROR_MS = 8000;   // error
```

Kullanıcının türden bağımsız olarak bir toast'ı erken kapatabilmesi için
her zaman bir kapatma düğmesi (`×`) render edilir.

## 2. Bir bileşenden toast gönderme

Yaygın durum: bir bileşen bir olayı işlemeyi bitirir ve onu tetikleyen
kullanıcıya başarıyı onaylamak (veya bir başarısızlığı raporlamak) ister.

```go
func (c *ContactForm) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
	switch event {
	// ...
	case "save":
		if !forms.ValidateAll(&c.Name, &c.Email, &c.Country, &c.Message, &c.Subscribe) {
			c.MarkDirty()
			return nil
		}
		c.Submitted = true
		c.ToastT("success", "contact.submit_success")
		c.MarkDirty()
	}
	return nil
}
```

Bu, `examples/contact-form/main.go` tarafından kullanılan tam desendir.
`ToastT`, `"contact.submit_success"`'ı bileşenin locale'inde (`c.Locale`,
oturum tarafından WS `?locale=` sorgu parametresinden ayarlanır) arar ve
onu `save` olayını gönderen aynı bağlantıya bir `success` toast'ı olarak
gönderir. Takip eden render (`session.sendRender`) tamamen ayrı bir
frame'dir — bir toast, normal diff/patch render döngüsünü asla
değiştirmez veya engellemez.

`Toast`/`ToastT`'yi `Mount`'tan, `HandleEvent`'ten, veya bileşenin
çağırdığı herhangi bir yardımcı metottan çağırabilirsiniz; her zaman
**bileşenin ait olduğu oturumu** hedefler, o an her hangi oturum olursa
olsun.

## 3. Bir bileşenin dışından push etme: `Hub`

Bazen bildirim, şu anda render eden bileşenle hiçbir ilgisi yoktur — bir
arka plan işi tamamlandı, başka bir kullanıcı bir yan etki tetikledi, bir
admin bir bakım bildirimi yayınlamak istiyor. Bunun için,
adapter'ınızı (`Options.Server`) bağlarken zaten oluşturduğunuz `*ws.Hub`'a
gidersiniz.

### 3.1 Hedeflenmiş: `Hub.Push`

```go
// Bir toast'ı ID'ye göre bir belirli oturuma gönderir.
if err := wsHub.Push(sessionID, ws.PushMessage{
	Kind: "info",
	Text: "Your export is ready.",
}); err != nil {
	// Oturum yoksa (grace period'ı geçmiş bağlantı kesilmişse) ws.ErrSessionNotFound.
	log.Printf("push failed: %v", err)
}
```

`Hub.Push`, oturumu `sessionID`'ye göre arar (istemcinin
`sessionStorage`'da kalıcı hale getirdiği ve yeniden bağlanmada geri
gönderdiği aynı ID) ve onun üzerinde `Session.EnqueuePush`'u çağırır.
Oturum artık kayıtlı değilse, `ws.ErrSessionNotFound` döndürür.

### 3.2 Broadcast: `Hub.Broadcast`

```go
// Bir toast'ı şu anda kayıtlı her oturuma gönderir.
wsHub.Broadcast(ws.PushMessage{
	Kind: "warning",
	Text: "Scheduled maintenance in 10 minutes.",
})
```

`Broadcast`, mevcut oturum listesinin bir okuma kilidi altında anlık
görüntüsünü alır ve her birinde push frame'ini kuyruğa alır. Asla
başarısız olmaz — kısa süreliğine (grace period'ları içinde) bağlantısı
kesilmiş oturumlar frame'i giden kanallarında basitçe tamponlar ve bir
bağlantı yeniden bağlandığında alır; dolu bir giden tamponu olan
oturumlar sessizce frame'i düşürür (aşağıdaki enqueue davranışına
bakın).

### 3.3 Gerçek bir örnek: admin broadcast route'u

`examples/contact-form/main.go`, bağlı her istemciye broadcast eden düz
bir HTTP uç noktası açığa çıkarır — herhangi bir ek UI oluşturmadan hızlı
bir "çevrimiçi olan herkese duyur" admin eylemi için kullanışlıdır:

```go
wsHub := ws.NewHub()
server := ws.NewServer(wsHub, registry, tr)
gouifiber.Register(app, gouifiber.Options{Server: server})

app.Get("/admin/broadcast", func(c fiber.Ctx) error {
	text := c.Query("text")
	if text == "" {
		text = "Sunucudan duyuru"
	}
	kind := c.Query("kind")
	if kind == "" {
		kind = "info"
	}
	wsHub.Broadcast(ws.PushMessage{Kind: kind, Text: text})
	return c.JSON(fiber.Map{"ok": true, "kind": kind, "text": text})
})
```

Çalışan örneğe karşı deneyin:

```
GET http://localhost:3001/admin/broadcast?text=Hello+everyone&kind=success
```

Şu anda GoUI WebSocket uç noktasına bağlı her tarayıcı sekmesi, sayfa
yeniden yükleme olmadan ve herhangi bir belirli bileşenin render
döngüsüyle ilişkisi olmadan hemen bir toast açar. Gerçek admin
araçları için kopyalanacak desen budur: route'u auth'un arkasına
gizleyin, ardından iş mantığınız bir bildirimin haklı olduğuna karar
verdiği her yerden (bir webhook handler'ı, bir cron job'u, başka bir
goroutine vb.) `hub.Broadcast`'i (veya tek bir kullanıcı için
`hub.Push`'u) çağırın.

## 4. İstemciyi bağlama

İstemci çalışma zamanı (`client/goui.js`), gelen `"push"` frame'lerini
zaten `GoUIClient`'i inşa ederken sağladığınız bir `onPush` callback'ine
yönlendirir. Bu callback'i toast modülünün `showToast` fonksiyonuna
bağlayın:

```js
import { GoUIClient } from '/client/goui.js';
import { enhanceToast, showToast } from '/client/modules/toast.js';

enhanceToast(); // <body> içinde .goui-toast-host konteynerini oluşturur

const client = new GoUIClient('/goui/ws', 'contact', {
  locale: 'en',
  onPush: showToast,
  onError: (msg) => console.error('[goui]', msg),
});

client.connect();
```

`enhanceToast(root)`, gerektiğinde ilk çağrıldığında tembel olarak tek
bir `<div class="goui-toast-host">`'u (`aria-live="polite"` ile)
oluşturur, dolayısıyla henüz bir toast ateşlenmeden başlangıçta bir kez
çağırmak güvenlidir. `showToast(payload)`:

1. `payload.kind`'i `success | error | warning | info`'dan (varsayılan
   `info`) birine normalize eder.
2. `payload.text` boşsa render etmeyi tamamen atlar.
3. Bir mesaj span'ı ve bir kapatma düğmesiyle bir
   `<div class="goui-toast goui-toast-<kind>">` oluşturur ve onu host'a
   **öne ekler** (en yeni üstte).
4. Hata toast'ları için `role="alert"`'i, diğer her şey için
   `role="status"`'u ayarlar, böylece ekran okuyucular onu uygun
   şekilde duyurur.
5. `error` için `ERROR_MS` (8000ms), diğer her şey için `DEFAULT_MS`
   (5000ms) sonrasında kaldırılmasını zamanlar — kapatma düğmesine
   tıklayarak iptal edilebilir.

## 5. Stilendirme

Toast renkleri, GoUI formlarının kalanının kullandığı aynı tasarım
token'larından gelir (bkz.
[12-theming-and-tailwind.md](12-theming-and-tailwind.md)). İlgili
kurallar, `forms/style.css` içinde `Toast / push notifications`
bölümünde yaşar:

```css
.goui-toast-host { position: fixed; top: 1rem; right: 1rem; z-index: 1000; /* ... */ }
.goui-toast { border: 1px solid var(--color-goui-border); background: var(--color-goui-surface); /* ... */ }
.goui-toast-success { border-color: ...var(--color-goui-success)...; background: ...var(--color-goui-success)...; }
.goui-toast-error   { border-color: ...var(--color-goui-error)...;   background: ...var(--color-goui-error)...;   }
.goui-toast-warning { border-color: ...var(--color-goui-warning)...; background: ...var(--color-goui-warning)...; }
.goui-toast-info    { border-color: ...var(--color-goui-info)...;    background: ...var(--color-goui-info)...;    }
```

Herhangi bir Go veya JS koduna dokunmadan toast'ları yeniden markalamak
için `--color-goui-*` özel özelliklerini kendi stil sayfanızda
(`forms/style.css`'ten sonra yüklenmiş) geçersiz kılın.

## 6. Bilmeniz gereken teslimat anlamları

- **Sunucuda gönder-ve-unut (fire-and-forget).** `EnqueuePush`, oturumun
  tamponlu giden kanalında (kapasite 32) bloklamayan bir gönderim yapar.
  Kanal doluysa — alışılmadık şekilde birikmiş bir istemci — frame,
  çağıranı bloklamak yerine sessizce düşürülür. Toast'lar geçici UX
  geri bildirimi içindir, garantili bir teslimat/denetim günlüğü değil.
- **Kısa bağlantı kesintilerine hayatta kalır.** Hedef oturumun
  bağlantısı kesikse ama hâlâ grace period'ı içindeyse (grace period
  detayları için [09-prefetch.md](09-prefetch.md) ve
  [13-project-integration.md](13-project-integration.md)'ye bakın), push
  frame'i kuyruğa girer ve WebSocket yeniden bağlanır bağlanmaz teslim
  edilir — okuyucu çevrimdışıyken "dinlemiyor" ama kanal tamponu frame'i
  tutar.
- **`Hub.Push` bir oturumu hedefler, bir bileşeni değil.** Bir oturum
  aynı anda birden fazla mount edilmiş/prefetch edilmiş bileşene ev
  sahipliği yapabilir; bir toast bunlardan hiçbirine kapsamlanmış
  değildir. Her zaman o tarayıcı sekmesi için üst seviye
  `.goui-toast-host`'ta ortaya çıkar.
- **Kalıcılık yok.** Toast'lar tasarım olarak geçicidir. Kullanıcıların
  çevrimdışıyken kaçırdıkları bir bildirimi görmesi gerekiyorsa, bunu
  bir toast olarak değil, uygulama verisi olarak modelleyin (örn. bir
  bileşen tarafından sonraki Mount'ta render edilen bir bildirim
  listesi).
</contents>
