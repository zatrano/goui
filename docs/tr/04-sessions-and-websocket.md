# 04 — Oturumlar, Hub, ve WebSocket Protokolü

Bu sayfa, `ws` paketini (`github.com/zatrano/goui/ws`) belgeler: `Session`/`Hub`
yaşam döngüsü, tel (wire) frame formatı, ve bunu `ws.Server` ile bir framework
adapter'ı aracılığıyla herhangi bir HTTP yığınına nasıl bağlayacağınız.

```go
import "github.com/zatrano/goui/ws"
```

## 1. Genel bakışta mimari

```
Browser (goui.js)
    │  event / prefetch / activate  (WebSocket üzerinden JSON frame'ler)
    ▼
Session ──► Component.HandleEvent / Mount
    │
    ▼
Render HTML ──► Diff (eski ağaç → yamalar) ──► Frame(render)
    │
    ▼
Hub (oturum map'i, grace-period yeniden bağlanma, Broadcast)
```

1. Tarayıcı `/goui/ws?component=<name>&locale=<locale>`'ye bağlanır.
2. Sunucu, `<name>`'i `core.Registry`'de arar, bir `Session` oluşturur,
   bileşeni mount eder, ve bir `session` frame'i ile ardından ilk `render`
   frame'ini (tam bir `OpReplace` yaması) gönderir.
3. Kullanıcı etkileşimi `event` frame'leri gönderir; sunucu `HandleEvent`'i
   çağırır, yeniden render eder, eski/yeni HTML'i diff'ler, ve minimal
   yama kümesini bir `render` frame'i olarak geri gönderir.
4. İsteğe bağlı olarak, `prefetch`, navigasyondan önce sessizce bir
   bileşeni mount eder; `activate` onu görünür kümeye yükseltir ve ilk
   render'ını gönderir.
5. Bağlantı kesildiğinde, oturum **hemen** sökülmez — bir grace period
   boyunca canlı tutulur, böylece aynı oturum ID'siyle yeniden bağlanma
   tam olarak kaldığı yerden devam eder.

## 2. `Session` yaşam döngüsü

```go
func NewSession(conn *websocket.Conn, translator *i18n.Translator, locale string) *Session
```

Bir `Session`, bir mantıksal tarayıcı sekmesi için oluşturulan her
`core.Component` örneğinin, tüm yaşam döngüsü boyunca — yeniden
bağlanmalar dahil — sahibidir. Metotlar aracılığıyla açığa çıkarılan
anahtar alanlar (aksi belirtilmedikçe hepsi private):

| Metot | Amaç |
|---|---|
| `SetRegistry(registry *core.Registry)` | `Prefetch`/`Activate` için isimleri çözmek üzere kullanılan registry'yi (ve dolaylı olarak ilk `?component=` bağlantısını) bağlar. `ws.Server.ServeConn` tarafından çağrılır; bunu nadiren kendiniz çağırırsınız. |
| `MountComponent(id string, c core.Component) error` | Çevirmeni/pusher'ı/ID'yi/Locale'i `c`'ye enjekte eder (`SetTranslator`, `SetPusher`, ve gömülü bir `BaseComponent`'e reflection aracılığıyla), sonra `c.Mount(ctx)`'i çağırır ve onu `id` altında saklar. İlk `?component=` sorgu parametresinden oluşturulan bileşen için kullanılır. |
| `Prefetch(name string) error` | `name`'i registry'de arar, taze bir örneği **render etmeden** mount eder, ve onu registry ismiyle (henüz oturuma görünür bir bileşen ID'si değil) anahtarlanmış bir yan map'te saklar. Aynı ismin yinelenen prefetch'leri no-op'tur. `ws.MaxPrefetch`'ten (5) fazla isim prefetch edildiğinde, en eskisi tahliye edilir (evict) ve `Unmount` edilir (ekleme sırasına göre basit LRU). |
| `Activate(name string) (string, error)` | Prefetch edilmiş bir bileşeni aktif kümeye yükseltir (ya da, hiç prefetch edilmediyse, taze bir tane oluşturur+mount eder), ona yeni bir bileşen ID'si atar, ilk tam render'ını gönderir, ve yeni ID'yi döndürür. |
| `Run(ctx context.Context)` | Okuma döngüsünü ve yazma döngüsünü iki goroutine olarak başlatır ve context iptal edilene, okuma döngüsü çıkana (bağlantı kapandı/hata), veya yazma döngüsü çıkana kadar bloklar. Döndüğünde, oturumu bağlantısı kesilmiş olarak işaretler ve altta yatan bağlantıyı kapatır — ama bileşenleri unmount **etmez**. WebSocket yükseltme başına bir kez, adapter'ınız yükseltilmiş bağlantıyı `ws.Server.ServeConn`'a verdikten sonra çağrılır. |
| `Reattach(conn *websocket.Conn) error` / `ReattachConn(conn wsConn) error` | Mevcut (bağlantısı kesilmiş) bir oturuma yeni bir canlı bağlantı bağlar, `disconnectedAt`'i temizler. Oturumun zaten canlı bir bağlantısı varsa (örn. eski bir yinelenmiş yeniden bağlanma denemesi) `ErrSessionAlreadyActive` döndürür. |
| `SendSessionFrame()` | Oturum ID'sini taşıyan bir `session` frame'i kuyruğa alır, böylece tarayıcı gelecekteki yeniden bağlanmalar için onu kalıcı hale getirebilir (`sessionStorage`). **Taze** bir bağlanmadan hemen sonra bir kez gönderilir (yeniden bağlanmada değil — istemci o zaman ID'sini zaten bilir). |
| `SendInitialRenders()` | Şu anda mount edilmiş her **aktif** bileşen için taze, tam bir `render` frame'i kuyruğa alır. Her bağlanma/yeniden bağlanmada çağrılır, böylece tarayıcının DOM'u sunucu durumuyla (yeniden) senkronize edilir — yeniden bağlanmanın istemci tarafında herhangi bir durum olmadan "sadece çalışmasını" sağlayan budur. |
| `Close() error` | Her aktif *ve* prefetch edilmiş bileşeni unmount eder (her birinde `Unmount(ctx)`), tüm dahili map'leri temizler, bağlantıyı kapatır, ve giden kanalı kapatır. Bağlantısı kesilmiş bir oturumun grace period'ı sona erdiğinde `Hub`'ın temizleme döngüsü tarafından çağrılır. |
| `IsDisconnected() bool` / `DisconnectedAt() time.Time` / `IsExpired(grace time.Duration) bool` | `Hub`'ın temizleme döngüsü tarafından kullanılan introspection; `IsExpired`, bağlıyken (`disconnectedAt` sıfırken) `false`'tur ve `time.Since(disconnectedAt) > grace` olduğunda `true` olur. |
| `EnqueuePush(msg PushMessage)` | Bir `push` frame'i kuyruğa alır. `BaseComponent.Toast`/`ToastT`'nin çağırdığı şey nihayetinde budur, her mount edilmiş bileşene `ws.Session.injectPusher`'ın enjekte ettiği pusher callback'i aracılığıyla. |

Dahili olarak, her `Session`'ın, yazma döngüsü tarafından tüketilen
tamponlu (buffered) bir giden kanalı (32 frame) vardır; kanal doluysa,
`enqueue` bloklamak yerine frame'i sessizce düşürür (yavaş/ölü bir
istemci okuma döngüsünü durdurmamalıdır).

## 3. `Hub` yaşam döngüsü

```go
type Hub struct { /* ... */ }

func NewHub() *Hub
func NewHubWithGracePeriod(grace time.Duration) *Hub

const DefaultGracePeriod = 60 * time.Second
```

`Hub`, canlı `Session`'ların process genelindeki registry'sidir, artı
arka planda çalışan bir temizleme goroutine'i.

- **`NewHub()`**, `DefaultGracePeriod` (`ws/hub.go`'da **`60 * time.Second`**
  olarak tanımlanmıştır) ile ve 10 saniyelik dahili bir temizleme
  aralığıyla (`defaultCleanupInterval`, şu anda yapılandırılamaz) bir hub
  başlatır.
- **`NewHubWithGracePeriod(grace time.Duration)`**, grace period'ı
  geçersiz kılmanıza olanak tanır — esas olarak temizleme davranışını
  deterministik olarak doğrulamak isteyen testler için tasarlanmıştır,
  ama 60s sizin yeniden bağlanma UX'inize uymuyorsa üretimde de
  ayarlamanızı önleyen hiçbir şey yoktur (örn. sallantılı bir mobil ağ
  daha uzun bir grace period isteyebilir; düşük bellekli çok kiracılı bir
  dağıtım daha kısa bir tane isteyebilir).

```go
hub := ws.NewHub()                              // 60s grace period
hub := ws.NewHubWithGracePeriod(10 * time.Second) // özel grace period
```

Hub metotları:

| Metot | Amaç |
|---|---|
| `Register(s *Session)` | Bir oturumu, `s.ID`'ye göre anahtarlanmış olarak hub'ın map'ine ekler. Taze her bağlanmada bir kez çağrılır (yeniden bağlanmada değil — oturum zaten var). |
| `Unregister(sessionID string)` | Bir oturumu kapatmadan map'ten kaldırır. Nadiren doğrudan çağrılır; temizleme `delete` + `Close`'u birlikte kullanır. |
| `Get(sessionID string) (*Session, bool)` | Bir oturumu ID'ye göre arar — WebSocket handler tarafından bir `?session=<id>` yeniden bağlanması için oturumu bulmak için kullanılır. |
| `Push(sessionID string, msg PushMessage) error` | Tam olarak bir oturuma, ID'ye göre bir push mesajı gönderir. ID kayıtlı değilse `ErrSessionNotFound` döndürür. |
| `Broadcast(msg PushMessage)` | Aynı push mesajını şu anda kayıtlı her oturuma gönderir — admin/bildirim uç noktalarının kullandığı budur (bkz. `contact-form` örneğinin `/admin/broadcast` route'u). |
| `Stop()` | Temizleme goroutine'ine çıkmasını sinyaller ve o çıkana kadar bloklar. Arka plan goroutine'inin düzgünce durmasını istiyorsanız uygulamanın düzgün kapanışında bunu çağırın (process'in çıkması için gerekli değildir, sadece tertiplilik için). |

Her 10 saniyede, temizleme döngüsü kayıtlı tüm oturumları tarar; `IsExpired(gracePeriod)`'u
`true` olan (grace period'dan daha uzun süre bağlantısı kesilmiş) her
oturum map'ten kaldırılır ve üzerinde `Close()` çağrılır (hâlâ sahip
olduğu her bileşeni unmount ederek).

## 4. Yeniden bağlanma anlamları (semantics)

- **Taze bağlanma:** `?component=<name>` (`?session=` yok) → registry
  araması, yeni `Session`, `MountComponent`, `hub.Register`,
  `SendSessionFrame`, `SendInitialRenders`, ardından `Run`.
- **Yeniden bağlanma:** `?session=<id>` → `hub.Get(id)`; bulunursa,
  `Reattach(conn)` canlı bağlantıyı yeniden bağlar (başka bir bağlantı
  zaten bağlıysa hata `ErrSessionAlreadyActive` — örn. aynı ID üzerinde
  yarışan iki sekme); yeni bir `session` frame'i gönderilmez (istemci
  ID'yi zaten bilir), ama bağlantısı kesikken gerçekleşen herhangi bir
  sunucu tarafı durum değişikliğine DOM'un yetişmesi için
  `SendInitialRenders()` hâlâ çalışır.
- **Bilinmeyen oturum:** `id`'nin hub'da olmadığı (zaten sona ermiş/
  temizlenmiş) bir `?session=<id>` → sunucu bir `error` frame'i yazar
  (`"session not found"`) ve bağlantıyı kapatır. İstemci tarafı çalışma
  zamanı (`goui.js`) bu mesajı özellikle tanır, saklanan oturum ID'sini
  `sessionStorage`'dan temizler, ve `?component=` ile yeniden taze
  bağlanır.

## 5. Frame protokolü referansı

WebSocket üzerindeki her mesaj, `ws.Frame` struct'ına uyan tek bir JSON
nesnesidir:

```go
type Frame struct {
    Type      string          `json:"type"`
    Component string          `json:"component,omitempty"`
    Event     string          `json:"event,omitempty"`
    Payload   json.RawMessage `json:"payload,omitempty"`
}
```

```go
const (
    FrameTypeEvent    = "event"
    FrameTypeRender   = "render"
    FrameTypePush     = "push"
    FrameTypeError    = "error"
    FrameTypeSession  = "session"
    FrameTypePrefetch = "prefetch"
    FrameTypeActivate = "activate"
)
```

| Tür | Yön | Payload şekli | Ne zaman gönderilir |
|---|---|---|---|
| `event` | istemci → sunucu | `{"component": "<id>", "event": "<name>", "payload": {...}}` | Her `g-click`/`g-change`/`g-submit`/debounce'lu-`g-input` etkileşimi. `Session.handleEventFrame`'e yönlendirilir; bu, bileşeni `component` (örnek ID'si) ile arar ve `component.HandleEvent(ctx, event, payload)`'ı çağırır. |
| `render` | sunucu → istemci | `[]diff.Patch` (yama nesnelerinin JSON dizisi: `op`, `path`, artı `html`, `text`, `attr`, `value`, `from_idx`, `to_idx` gibi op'a özgü alanlar) | `HandleEvent`, `core.ErrSkipRender` döndürmeden başarılı olduktan sonra (önceki ağaca karşı artımlı yama), bir bileşenin çok ilk render'ında (tam `OpReplace`), ve her (yeniden) bağlanmada `SendInitialRenders`'tan bileşen başına bir kez. |
| `push` | sunucu → istemci | `ws.PushMessage` — `{"kind": "success\|error\|warning\|info", "text": "..."}` | `BaseComponent.Toast`/`ToastT` çağrıldığında (oturum başına) veya `Hub.Broadcast`/`Hub.Push` çağrıldığında (admin/sistem bildirimleri). Belirli bir bileşene bağlı değildir — oturum genelinde bir bildirimdir. |
| `error` | sunucu → istemci | `ws.ErrorPayload` — `{"message": "..."}` | Bozuk gelen JSON, bir `event` frame'inde bilinmeyen bileşen ID'si, bir `HandleEvent`/`Render`/diff hatası, başarısız bir `Reattach`, veya bağlanmada bilinmeyen/süresi dolmuş bir `?session=` ID'si. |
| `session` | sunucu → istemci | `ws.SessionPayload` — `{"id": "<session-id>"}` | **Taze** bir bağlanmadan hemen sonra tam olarak bir kez (yeniden bağlanmada asla). İstemci bu ID'yi (`sessionStorage`) kalıcı hale getirir ve gelecekteki yeniden bağlanma denemelerinde `?session=` olarak dahil eder. |
| `prefetch` | istemci → sunucu | `{"component": "<registry-name>"}` (not: `Component` burada *registry ismini* tutar, henüz bir örnek ID'si değil) | Bir `data-goui-prefetch="<name>"` içeren bir eleman üzerinde imleç geçişinde (~100ms) veya viewport'a girişte `prefetch.js` istemci modülü tarafından gönderilir. `Session.Prefetch(name)`'i tetikler — sessizce mount eder, geri render gönderilmez. |
| `activate` | istemci → sunucu | `{"component": "<registry-name>"}` | `data-goui-activate="<name>"` içeren bir elemana tıklandığında `prefetch.js` tarafından gönderilir. `Session.Activate(name)`'i tetikler; bu, prefetch edilmiş örneği yükseltir (veya prefetch edilmediyse taze mount eder) ve — taze üretilmiş bir bileşen örneği ID'si ile — ilk `render` frame'ini hemen gönderir. |

Notlar:

- `prefetch`/`activate` frame'leri `component` alanında **registry ismini**
  taşır (örn. `"contact"`), oysa `event`/`render` frame'leri bir **çalışma
  zamanı örneği ID'sini** (mount edilmiş her bileşen için üretilen rastgele
  bir hex string) taşır. İkisini karıştırmayın — bir registry ismi her
  seferinde yepyeni bir örnek ID'sine activate edilebilir.
- `render` payload'ları her zaman bir `diff.Patch` dizisidir, bir
  bileşenin çok ilk render'ı için bile
  (`{"op": "replace", "path": [], "html": "...", "tag": "..."}` şeklinde
  tek elemanlı bir dizi), dolayısıyla "ilk render" ve "artımlı yama" için
  istemci tarafı mantığı birleştirilmiştir.
- `HandleEvent` içindeki, tam olarak `core.ErrSkipRender` olan hatalar
  (`errors.Is` ile) hiçbir frame üretmez — bir `error` frame'i bile.
  Bu, istemciye ait editörler için kasıtlıdır; bkz.
  [`07-forms-tier2.md`](07-forms-tier2.md).

## 6. `ws.Server`, adapter'lar ve `main.go` iskeleti

Çekirdek modül framework-agnostic bir acceptor sunar:

```go
type Server struct { /* Hub, Registry, Translator */ }

func NewServer(hub *Hub, registry *core.Registry, translator *i18n.Translator) *Server

// Path varsayılan WebSocket uç noktası yoludur.
const Path = "/goui/ws"

type ConnectParams struct {
    SessionID     string // ?session=
    ComponentName string // ?component=
    Locale        string // ?locale=
}

func (s *Server) ServeConn(ctx context.Context, conn Conn, p ConnectParams) error
```

Her adapter HTTP WebSocket yükseltmesini yapar, yukarıdaki sorgu parametrelerini
okur ve `ServeConn`'u çağırır. Özel bir adapter yazmadıkça `ServeConn`'u
kendiniz çağırmazsınız.

Route'ları yığınınıza uygun adapter üzerinden kaydedin (`Options.Server`
gerekli; `Options.Store` isteğe bağlı — bkz.
[11-file-uploads.md](11-file-uploads.md)):

```go
server := ws.NewServer(hub, registry, tr)

// Fiber
import gouifiber "github.com/zatrano/goui/adapters/fiber"
gouifiber.Register(app, gouifiber.Options{Server: server, Store: store})

// net/http ServeMux
import gouistdlib "github.com/zatrano/goui/adapters/stdlib"
gouistdlib.Register(mux, gouistdlib.Options{Server: server, Store: store})

// Chi (stdlib adapter Mount)
gouistdlib.Mount(chiRouter, gouistdlib.Options{Server: server})

// Gin
import gouigin "github.com/zatrano/goui/adapters/gin"
gouigin.Register(r, gouigin.Options{Server: server, Store: store})

// Echo
import gouiecho "github.com/zatrano/goui/adapters/echo"
gouiecho.Register(e, gouiecho.Options{Server: server, Store: store})
```

Her şeyi birbirine bağlayan Fiber `main.go` iskeleti:

```go
package main

import (
    "log"
    "path/filepath"

    "github.com/gofiber/fiber/v3"
    "github.com/gofiber/fiber/v3/middleware/static"

    gouifiber "github.com/zatrano/goui/adapters/fiber"
    "github.com/zatrano/goui/core"
    "github.com/zatrano/goui/i18n"
    "github.com/zatrano/goui/upload"
    "github.com/zatrano/goui/ws"
)

func main() {
    // 1. i18n
    tr := i18n.NewTranslator()
    _ = tr.LoadLocale("tr", filepath.Join("i18n", "locales", "tr.json"))
    _ = tr.LoadLocale("en", filepath.Join("i18n", "locales", "en.json"))

    // 2. Bileşen registry'si
    registry := core.NewRegistry()
    if err := registry.Register("counter", func() core.Component { return &Counter{} }); err != nil {
        log.Fatal(err)
    }

    // 3. Hub — varsayılan 60s grace period, veya ayarlayın:
    hub := ws.NewHub()
    // hub := ws.NewHubWithGracePeriod(30 * time.Second)
    server := ws.NewServer(hub, registry, tr)

    // 4. Fiber uygulaması + statik varlıklar
    app := fiber.New()
    app.Use("/client", static.New("./client"))
    app.Get("/", func(c fiber.Ctx) error {
        return c.SendFile("index.html")
    })

    // 5. (isteğe bağlı) dosya yüklemeleri — bkz. 11-file-uploads.md
    store, err := upload.NewLocalStore("./.goui-uploads", "/goui/files", 8<<20)
    if err != nil {
        log.Fatal(err)
    }

    // 6. GoUI route'ları (WebSocket + isteğe bağlı upload)
    gouifiber.Register(app, gouifiber.Options{Server: server, Store: store})

    // 7. (isteğe bağlı) sunucu başlatmalı push, örn. bir admin broadcast uç noktası
    app.Get("/admin/broadcast", func(c fiber.Ctx) error {
        hub.Broadcast(ws.PushMessage{Kind: "info", Text: c.Query("text", "Hello")})
        return c.JSON(fiber.Map{"ok": true})
    })

    log.Println("listening on http://localhost:3000")
    log.Fatal(app.Listen(":3000"))

    // Kodunuzun başka bir yerinde düzgün kapanışta: hub.Stop()
}
```

`Counter` bileşeninin kendisi için [`01-getting-started.md`](01-getting-started.md)'ye,
ve gerçek bir uygulamada `counter` yerine ne kaydedeceğiniz için
[`05-forms-tier1.md`](05-forms-tier1.md) / [`07-forms-tier2.md`](07-forms-tier2.md)'ye
bakın.
</contents>
