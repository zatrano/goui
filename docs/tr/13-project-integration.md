# 13. Proje Entegrasyonu

Bu, bu belgelerdeki en önemli bölümdür: GoUI'yi gerçek, zaten var olan bir
HTTP uygulamasına düşürmeyi baştan sona anlatır — routing, adapter seçimi,
çok kiracılılık (multi-tenancy), PostgreSQL, sıfırdan yeni bir bileşen
inşa etme, gerçekçi bir çok-katmanlı form kompoze etme, ve Docker
olmadan üretimde çalıştırma.

Bu belge boyunca kullanılan modül yolu: `github.com/zatrano/goui`.

```go
import (
	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/forms"
	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/upload"
	"github.com/zatrano/goui/validation"
	"github.com/zatrano/goui/ws"
)
```

---

## 1. Mevcut bir HTTP uygulamasına entegre etme

GoUI kendi HTTP sunucusunu veya kendi uygulama struct'ını istemez — sizin
zaten sahip olduğunuz yönlendiriciye, uygulamanızın zaten sunduğu diğer
route'ların yanına (REST API'leri, sunucu tarafında render edilen
sayfalar, health check'ler vb.) bir avuç route kaydetmek ister. Bağlanması
gereken tam olarak üç şey vardır:

1. **Statik varlıklar** — istemci çalışma zamanı (`client/goui.js` ve
   `client/modules/*.js`) ve temel stil sayfası (`forms/style.css`) düz
   dosyalardır; framework'ünüzün statik dosya middleware'i veya
   `http.FileServer` ile sunun.
2. **WebSocket uç noktası** — `ws.NewServer(hub, registry, tr)` oluşturun
   ve `GET /goui/ws` (`ws.Path`) yolunu bir adapter üzerinden bağlayın.
3. **Yükleme uç noktaları** (sadece `forms.DragDropUpload` /
   `forms.AvatarUpload` kullanıyorsanız) — adapter'ın `Store` seçeneğine
   bir `upload.Storage` verin veya `net/http` mux üzerinde `upload.Mount`
   çağırın.

### 1.1 Adapter seçimi

| Yığın | Modül | Kayıt |
|---|---|---|
| Fiber v3 | `github.com/zatrano/goui/adapters/fiber` | `gouifiber.Register(app, opts)` |
| `net/http` | `github.com/zatrano/goui/adapters/stdlib` | `gouistdlib.Register(mux, opts)` |
| Chi | `github.com/zatrano/goui/adapters/stdlib` | `gouistdlib.Mount(chiRouter, opts)` |
| Gin | `github.com/zatrano/goui/adapters/gin` | `gouigin.Register(r, opts)` |
| Echo | `github.com/zatrano/goui/adapters/echo` | `gouiecho.Register(e, opts)` |

Her adapter aynı seçenek yapısını kabul eder:

```go
type Options struct {
    Server *ws.Server      // WebSocket için gerekli
    Store  upload.Storage  // isteğe bağlı — POST /goui/upload ve GET /goui/files/:id
}
```

Kanıt örnekleri: `examples/adapters/{nethttp,chi,gin,echo}`. `examples/`
altındaki ana demolar Fiber adapter'ını kullanır.

### 1.2 Yığın bazında parçalar

**Fiber** (repodaki demolar da bunu kullanır):

```go
import gouifiber "github.com/zatrano/goui/adapters/fiber"

hub := ws.NewHub()
server := ws.NewServer(hub, registry, tr)
store, _ := upload.NewLocalStore("./data/uploads", "/goui/files", 8<<20)

gouifiber.Register(app, gouifiber.Options{Server: server, Store: store})
```

**net/http `ServeMux`:**

```go
import gouistdlib "github.com/zatrano/goui/adapters/stdlib"

gouistdlib.Register(mux, gouistdlib.Options{
    Server: ws.NewServer(hub, registry, tr),
    Store:  store,
})
```

**Chi** (stdlib adapter `Mount` ile):

```go
import gouistdlib "github.com/zatrano/goui/adapters/stdlib"

gouistdlib.Mount(chiRouter, gouistdlib.Options{
    Server: ws.NewServer(hub, registry, tr),
    Store:  store,
})
```

**Gin:**

```go
import gouigin "github.com/zatrano/goui/adapters/gin"

gouigin.Register(r, gouigin.Options{
    Server: ws.NewServer(hub, registry, tr),
    Store:  store,
})
```

**Echo:**

```go
import gouiecho "github.com/zatrano/goui/adapters/echo"

gouiecho.Register(e, gouiecho.Options{
    Server: ws.NewServer(hub, registry, tr),
    Store:  store,
})
```

### 1.3 Tam Fiber entegrasyon örneği

Aşağıdaki örnek tipik bir mevcut Fiber v3 uygulamasına uyar; farklı bir
yığın kullanıyorsanız `gouifiber.Register`'ı §1.2'deki adapter'ınızla
değiştirin.

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

	"myapp/internal/httpapi" // mevcut REST handler'larınız, GoUI'yle ilgisiz
)

func main() {
	app := fiber.New()

	// --- mevcut route'larınız, dokunulmamış ---
	app.Use(myAuthMiddleware())
	httpapi.RegisterRoutes(app)

	// --- GoUI bağlantısı bunların yanında yaşar ---
	gouiRoot := "./vendor/goui" // goui modülünün client+forms varlıklarını vendor'ladığınız/checkout ettiğiniz her yer
	app.Use("/client", static.New(filepath.Join(gouiRoot, "client")))
	app.Use("/forms", static.New(filepath.Join(gouiRoot, "forms"))) // forms/style.css

	tr := i18n.NewTranslator()
	_ = tr.LoadLocale("en", "./locales/en.json")
	_ = tr.LoadLocale("tr", "./locales/tr.json")

	registry := core.NewRegistry()
	mustRegisterComponents(registry, tr /*, db, vb. */)

	hub := ws.NewHub()
	server := ws.NewServer(hub, registry, tr)

	store, err := upload.NewLocalStore("./data/uploads", "/goui/files", 8<<20)
	if err != nil {
		log.Fatal(err)
	}

	gouifiber.Register(app, gouifiber.Options{Server: server, Store: store})

	// Tarayıcıda bir GoUI bileşenini başlatan bir sayfa, sadece bir
	// <script type="module"> bootstrap'ı ile HTML sunan normal bir
	// handler'dır — bootstrap script'inin kendisi için §5'teki
	// compose-form örneğine bakın.
	app.Get("/reseller/register", func(c fiber.Ctx) error {
		return c.SendFile("./views/reseller_register.html")
	})

	log.Fatal(app.Listen(":8080"))
}
```

Çalışma zamanında GoUI'nin ihtiyaç duyduğu her şey: doldurulmuş bir
`*core.Registry`, bir `*ws.Server`, isteğe bağlı `upload.Storage`, statik
varlık route'ları ve tek bir adapter `Register`/`Mount` çağrısıdır. GoUI
hakkında hiçbir şey, `/goui/*` ve `/client` ile `/forms` için seçtiğiniz
statik öneklerin ötesinde kendi process'ini, portunu veya ters proxy
yolunu gerektirmez.

> `/client` ve `/forms` önekleri hakkında not: bunlar sizin
> seçiminizdir — bu dizinleri HTTP yığınınızın statik dosya sunma
> yöntemiyle bağlayın. Bunları seçtikten sonra sabit tutun, çünkü
> sunduğunuz HTML ve `input.css`'in `@import` yolu
> ([12-theming-and-tailwind.md](12-theming-and-tailwind.md)'ye bakın)
> ikisi de onlarla uyuşmalıdır.

---

## 2. Çok kiracılı desen — Session koduna dokunmadan

Çok yaygın gerçek dünya şekli: **Holding → Şirket → Departman →
Kullanıcı** gibi bir organizasyonel hiyerarşi, ki her dashboard/bileşen
sadece giriş yapmış kullanıcının ait olduğu kiracıya kapsamlanmış veriyi
görmeli ve değiştirmelidir. GoUI'nin `ws.Session`'ı kasıtlı olarak
**hiçbir kiracılık kavramı** içermez — ve bunu `ws/session.go`'yu
değiştirerek eklememelisiniz. Session'ın işi taşımadır (frame'ler,
yeniden bağlanma, prefetch defter tutumu); kiracılık uygulama
verisidir, ve bileşenlerinizde yaşamalıdır, taşımada değil.

### 2.1 `context.Context`'in kendisi neden bunu taşımaz

`Mount(ctx context.Context)` içinde `ctx`'e uzanmak ve kendi Fiber
middleware'inizin `c.Locals("tenantID")`'sinin bir şekilde orada
olmasını beklemek cazip gelir. Orada olmayacaktır:
`Session.prepareComponent`, her zaman `c.Mount(context.Background())`'u
çağırır — WS yükseltme el sıkışması, sayfa *önyükleyen* WebSocket'i
sunulduğunda var olan herhangi bir HTTP istek bağlamından bağımsız,
kendi bağlantı yaşam döngüsünde gerçekleşir. "Uygulamanızın kimliği
doğrulanmış HTTP isteği" ile "bir GoUI bileşeni örneği" arasında tam
olarak bir hand-off noktası vardır, ve bu, tarayıcı soketi açtığında
istenen **registry bileşen ismidir** (WS URL'sindeki
`?component=<name>`, veya `data-goui-prefetch`/`data-goui-activate`
öznitelik değeri). Çok kiracılığınızı bu hand-off noktası etrafında
tasarlayın.

### 2.2 Desen: kiracı-nitelikli registry isimleri + factory closure'ları

İki bileşen:

1. **Kiracı kapsamı başına bir factory kaydedin**, kiracı ID'lerini (ve
   bileşenin ihtiyaç duyduğu herhangi bir DB handle'ını/servisi)
   closure değişkenleri olarak yakalayarak — bu tam olarak "factory'de
   ayarlanmış struct alanlarında kiracı ID'sini saklama" yaklaşımıdır:
   `registry.Create(name)`'den geri gelen `Component` örneği, `Mount`
   hiç çağrılmadan önce, zaten doldurulmuş `HoldingID`/`CompanyID`/
   `DeptID`/`UserID`'ye sahiptir.
2. **Kiracı kapsamını registry isminin kendisine kodlayın**, ve kendi
   (mevcut, değiştirilmemiş) kimlik doğrulanmış sayfa handler'ınızın o
   ismi sunduğu bootstrap script'ine render etmesini sağlayın — bu,
   "Mount'tan önce middleware'inizin ayarladığı bağlam değerleri"
   kısmıdır: middleware'iniz *sayfa* isteğinde çalışır, WS yükseltmeden
   çok önce, ve onun kararı (hangi kiracı), tarayıcının isteyeceği
   isme pişirilir.

```go
// internal/dashboard/component.go
package dashboard

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zatrano/goui/core"
)

// DeptDashboard, inşa zamanında tam olarak bir departmana kapsamlanmıştır.
// ws.Session veya ws.Hub hakkında hiçbir şeyin bu alanların var olduğunu bilmesi gerekmez.
type DeptDashboard struct {
	core.BaseComponent

	db *pgxpool.Pool

	HoldingID string
	CompanyID string
	DeptID    string
	UserID    string

	Employees []employeeRow
}

type employeeRow struct {
	ID, FullName, Role string
}
```

Kiracıya kapsamlanmış factory'leri **kendi** kimlik doğrulanmış istek
handler'ınızdan tembel ve idempotent olarak kaydetme (`ws`'e veya
`core`'a değişiklik gerekmez — `Register`, basitçe yinelenen bir ismi
reddeder):

```go
// internal/dashboard/register.go
package dashboard

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zatrano/goui/core"
)

// componentName, §2.1'de açıklanan registry-ismi hand-off noktasını inşa eder.
// Kiracı yolunun herhangi bir deterministik, çakışmasız kodlaması işe yarar —
// bu sadece "dept-dashboard:<deptID>"dir.
func componentName(deptID string) string {
	return fmt.Sprintf("dept-dashboard:%s", deptID)
}

// EnsureRegistered, belirli bir departman ilk görüldüğünde tembel olarak
// kiracıya kapsamlanmış bir factory kaydeder. Her istekte çağırmak güvenlidir;
// iki kez kaydetmek no-op'tur (ErrComponentAlreadyRegistered yutulur).
func EnsureRegistered(registry *core.Registry, db *pgxpool.Pool, holdingID, companyID, deptID, userID string) string {
	name := componentName(deptID)
	err := registry.Register(name, func() core.Component {
		return &DeptDashboard{
			db:        db,
			HoldingID: holdingID,
			CompanyID: companyID,
			DeptID:    deptID,
			UserID:    userID,
		}
	})
	if err != nil && !errors.Is(err, core.ErrComponentAlreadyRegistered) {
		panic(err) // "already registered" olmayan kayıt hataları bir hata belirtir
	}
	return name
}
```

Mevcut kimlik doğrulanmış sayfa handler'ınız (`EnsureRegistered`'ı
çağırmak ve ortaya çıkan ismi şablona geçirmek dışında dokunulmamış),
kiracılık ve GoUI'nin gerçekten buluştuğu tek yerdir:

```go
app.Get("/dashboard", requireLogin(), func(c fiber.Ctx) error {
	user := currentUser(c) // mevcut auth'unuz, GoUI'yle ilgisiz
	name := dashboard.EnsureRegistered(registry, db,
		user.HoldingID, user.CompanyID, user.DeptID, user.ID)

	return c.Render("dashboard", fiber.Map{
		"ComponentName": name, // örn. "dept-dashboard:D-42"
	})
})
```

```html
<!-- dashboard.html, ComponentName enjekte edilerek sunucu tarafında render edilmiş -->
<div id="app"></div>
<script type="module">
  import { GoUIClient } from '/client/goui.js';
  const client = new GoUIClient('/goui/ws', '{{.ComponentName}}', { mount: '#app', locale: 'en' });
  client.connect();
</script>
```

Tarayıcı soketi `?component=dept-dashboard:D-42` ile açtığında,
`registry.Create`, *o belirli departmanın* factory closure'ını çalıştırır
— ortaya çıkan bileşen, `ws/session.go`, `ws/hub.go`, veya
`ws/server.go`'ya **sıfır değişiklikle**, hangi holding, şirket,
departman, ve kullanıcıya ait olduğunu zaten bilir. `Mount` ve
`HandleEvent`, ardından SQL parametreleri olarak basitçe
`d.HoldingID`/`d.CompanyID`/`d.DeptID`/`d.UserID`'yi kullanır (§3) —
kiracılık için `ctx`'e asla danışmaları gerekmez, çünkü zaten bir
alandır.

### 2.3 Anlatı özeti: Holding → Şirket → Departman → Kullanıcı

- **Holding** — üst seviye tüzel kişilik; birçok Şirkete sahiptir.
- **Şirket** — bir Holding altındaki bir tüzel/operasyonel işletme
  birimi; birçok Departmana sahiptir.
- **Departman** — bir Şirket altında organizasyonel bir birim (Satış,
  Destek, Mühendislik, ...); birçok Kullanıcısı vardır ve çoğu günlük
  dashboard için doğal "çalışma alanı" kapsamıdır.
- **Kullanıcı** — (bu basitleştirilmiş modelde) tam olarak bir
  Departman içinde bir rolü olan bir birey — gerçek sistemler genellikle
  departmanlar arası rollere izin verir, ki bu sadece aynı closure'da
  daha fazla ID yakalamak anlamına gelir.

Bu hiyerarşinin her seviyesi, factory closure'ı tarafından yakalanan
başka bir alan ve SQL'inizde (§3) başka bir `WHERE` cümlesi
parametresidir. Belirli bir görünüm için mantıklı olan herhangi bir
seviyede bileşen kaydedebilirsiniz — sadece `CompanyID` ile
anahtarlanmış bir şirket genelinde bir özet dashboard'u, `DeptID` ile
anahtarlanmış bir departman dashboard'u, veya `UserID` ile
anahtarlanmış kişisel bir "görevlerim" bileşeni — desen her seviyede
aynıdır; sadece registry-ismi kodlaması ve SQL `WHERE` cümlesi
değişir.

### 2.4 Kayıtları temizleme

`*core.Registry`'nin bir `Unregister`'ı yoktur; girdiler process'in
yaşam döngüsü boyunca birikir. Büyük veya sınırsız sayıda departmanı
olan uzun süre çalışan çok kiracılı bir sunucu için, belirli departman
ID'sini bağlantı başına stabil başka bir şeyden okuyan **rol seviyesi**
isimlerini (`"dept-dashboard"`) kaydetmeyi tercih edin — bileşen ismine
gömülü, kısa ömürlü, imzalanmış bir token gibi (örn.
`"dept-dashboard:" + signedDeptToken`), bileşeni döndürmeden önce
factory içinde doğrulanmış ve çözülmüş — her departman için sonsuza
kadar bir literal isim kaydetmek yerine. Bunun eklenen karmaşıklığa
değip değmediği tamamen kiracı kardinalitenize bağlıdır; onlarca veya
düşük yüzlerce departman için, yukarıda gösterilen basit tembel-kayıt
yeterince basit ve tamamen uygundur.

---

## 3. PostgreSQL: `Mount`'ta okuma, `HandleEvent`'te yazma

`DeptDashboard` örneğine devam ederek — bu SQL örnektir (kendi
şemanıza/sürücünüze göre ayarlayın), ama şekil — `Mount`'ta okuma,
bir şeyi gönderen/değiştiren olayda doğrula+yaz — gerçek bir
veritabanıyla desteklenen herhangi bir bileşen için takip edilecek
desendir.

```go
func (d *DeptDashboard) Mount(ctx context.Context) error {
	rows, err := d.db.Query(ctx, `
		SELECT id, full_name, role
		FROM employees
		WHERE holding_id = $1 AND company_id = $2 AND dept_id = $3
		ORDER BY full_name
	`, d.HoldingID, d.CompanyID, d.DeptID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var e employeeRow
		if err := rows.Scan(&e.ID, &e.FullName, &e.Role); err != nil {
			return err
		}
		d.Employees = append(d.Employees, e)
	}
	return rows.Err()
}

func (d *DeptDashboard) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
	switch event {
	case "promote":
		employeeID, _ := payload["value"].(string)
		if employeeID == "" {
			return nil
		}
		// Her yazmayı her zaman inşa zamanında yakalanan kiracı alanlarıyla
		// yeniden kapsamlayın — kiracılık için payload'dan hiçbir şeye asla güvenmeyin.
		tag, err := d.db.Exec(ctx, `
			UPDATE employees
			SET role = 'lead'
			WHERE id = $1 AND holding_id = $2 AND company_id = $3 AND dept_id = $4
		`, employeeID, d.HoldingID, d.CompanyID, d.DeptID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			// ID mevcut değil, veya (daha da önemlisi) farklı bir kiracıya ait
			// — ikisini de aynı şekilde ele alın: no-op, sızıntı yok.
			d.ToastT("error", "dashboard.promote_denied")
			return nil
		}
		for i := range d.Employees {
			if d.Employees[i].ID == employeeID {
				d.Employees[i].Role = "lead"
			}
		}
		d.ToastT("success", "dashboard.promote_ok")
		d.MarkDirty()
	}
	return nil
}
```

Kritik güvenlik özelliği: **her yazmanın `WHERE` cümlesi, sadece satırın
birincil anahtarını değil, kiracı alanlarını da tekrarlar.** Bu alanlar
factory closure'ından (§2.2) geldiğinden, WS olay payload'ından değil,
kötü niyetli bir istemci, gönderdiği JSON'u kurcalayarak kendi kapsamını
genişletemez — yapabileceği en kötü şey, kendi departmanına ait olmayan
bir `employeeID` adlandırmaktır, ki bu, `RowsAffected() == 0` kontrolünü
zararsız bir no-op'a çevirir, çapraz-kiracı bir mutasyona değil.

---

## 4. Adım adım yeni bir bileşen inşa etme

1. **Bileşenin hangi durumun sahibi olduğuna karar verin.** Bunu tam
   olarak `Render()`'ın ihtiyaç duyduğu ve `HandleEvent`'in mutasyona
   uğrattığı şeye indirin — başka bir şey değil.

2. **`MarkDirty`/`IsDirty`/`ResetDirty`, `T`/`ToastT`, ve oturumun
   otomatik olarak bağladığı çevirmen/pusher defter tutumu için
   `core.BaseComponent`'i gömün.**

3. **Dört `core.Component` metodunu uygulayın:**

   ```go
   type Counter struct {
       core.BaseComponent
       Value int
       Step  int
   }

   func NewCounter(step int) *Counter {
       if step == 0 {
           step = 1
       }
       return &Counter{Step: step}
   }

   func (c *Counter) Mount(_ context.Context) error   { return nil } // yüklenecek harici durum yok
   func (c *Counter) Unmount(_ context.Context) error { return nil } // serbest bırakılacak bir şey yok

   func (c *Counter) HandleEvent(_ context.Context, event string, _ map[string]any) error {
       switch event {
       case "inc":
           c.Value += c.Step
       case "dec":
           c.Value -= c.Step
       default:
           return nil // bilinmeyen olaylar yoksayılır, hata değil
       }
       c.MarkDirty()
       return nil
   }

   func (c *Counter) Render() (string, error) {
       return fmt.Sprintf(
           `<div data-goui-ignore="false"><span>%d</span> `+
               `<button type="button" g-click="dec">-</button> `+
               `<button type="button" g-click="inc">+</button></div>`,
           c.Value,
       ), nil
   }
   ```

   [10-diffing-internals.md](10-diffing-internals.md) §3'ten
   hatırlayın: `Render()`, GoUI'nin sentetik bir sarmalayıcı `<div>`
   enjekte etmesi asla gerekmesin diye **tam olarak bir kök eleman**
   döndürmelidir.

4. **Bir örnek değil, bir factory kaydedin:**

   ```go
   registry.Register("counter", func() core.Component { return NewCounter(1) })
   ```

   `Register`, yinelenen bir isimde `core.ErrComponentAlreadyRegistered`
   ile başarısız olur ve `Create`, bilinmeyen bir isim için
   `core.ErrComponentNotRegistered` ile başarısız olur — dinamik olarak
   kaydettiğiniz/activate ettiğiniz yerde bunları kontrol edin (§2.4'te
   olduğu gibi).

5. **Bileşen çevrilmiş metne veya toast'lara ihtiyaç duyuyorsa**,
   inşa sırasında `c.SetTranslator(tr)`'yi çağırın (veya factory'nin
   `tr`'yi kapatmasına (close over) izin verin ve döndürmeden önce
   ayarlayın) — `Session.prepareComponent`, her `Mount`/`Activate`'te
   otomatik olarak `SetTranslator`/`SetPusher`'ı da çağırır, dolayısıyla
   bu adım çoğunlukla bileşeni bir oturum olmadan doğrudan inşa eden
   testlerde bile çevrilmiş string'lerin mevcut olmasını istiyorsanız
   ilgilidir.

6. **Kompoze HTML'i bağlayın.** Önemsizin ötesindeki her şey için, düz
   string birleştirme/`strings.Builder` ile (`forms/*` boyunca
   kullanılan konvansiyon) inşa edin, veya daha şablon-benzeri
   ergonomi için `core.RenderTemplate`'i kullanın:

   ```go
   html, err := core.RenderTemplate(`
     <div>
       <span>{{.Value}}</span>
       <button type="button" g-click="dec">-</button>
       <button type="button" g-click="inc">+</button>
     </div>`, c)
   ```

   `RenderTemplate`, ayrıştırılmış şablonları şablon string'inin bir
   hash'i ile önbelleğe alır, dolayısıyla onu her `Render()`'da aynı
   literal şablon metniyle çağırmak her çağrıda yeniden ayrıştırma
   yapmaz.

7. **`HandleEvent` → `Render`'ı doğrudan, hiç `Session` olmadan
   test eden bir test yazın** — bileşenler, WS katmanına gizli bir
   bağımlılığı olmayan düz Go değerleridir:

   ```go
   func TestCounter_Increment(t *testing.T) {
       c := NewCounter(1)
       if err := c.HandleEvent(context.Background(), "inc", nil); err != nil {
           t.Fatal(err)
       }
       if !c.IsDirty() {
           t.Fatal("expected dirty after inc")
       }
       html, err := c.Render()
       if err != nil || !strings.Contains(html, ">1<") {
           t.Fatalf("html = %q, err = %v", html, err)
       }
   }
   ```

8. **Tarayıcıya registry ismini vererek** (doğrudan, veya §2'deki
   kiracı-nitelikli desen aracılığıyla) onu bir sayfada mount edin, ve
   ikincil bir görünümse, düz bir link yerine `data-goui-prefetch`/
   `data-goui-activate`'i ([09-prefetch.md](09-prefetch.md))
   düşünün.

---

## 5. Compose form: Tier 1 + Tier 2 + doğrulama — "3CX bayi kaydı"

Düz alanları (Tier 1: `TextInput`, `Select`, `ChoiceInput`), kompoze
edilmiş bir Tier 2 kontrolünü (`forms.PhoneInput`, ki bu kendisi bir
`SearchableSelect` çevirme kodu seçicisi artı bir `TextInput`'ü bağlar),
ve `forms.ValidateAll` aracılığıyla sunucu tarafı doğrulamayı birleştiren
gerçekçi bir örnek — `examples/contact-form`'un kullandığı aynı desen,
daha fazla alan ve çapraz alan kaygılarıyla ölçeklendirilmiş.

```go
package reseller

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/forms"
	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/validation"
)

// RegistrationForm bir "3CX bayisi olma" başvuru formudur:
// şirket kimliği, iletişim kanalı, ve program katmanı seçimi.
type RegistrationForm struct {
	core.BaseComponent

	db *pgxpool.Pool

	CompanyName forms.TextInput
	Website     forms.TextInput
	Email       forms.TextInput
	Phone       *forms.PhoneInput
	Country     forms.SearchableSelect
	Tier        forms.Select
	MonthlySeats forms.NumericInput
	AgreeTerms  forms.ChoiceInput

	Submitted bool
	Summary   string
}

func NewRegistrationForm(db *pgxpool.Pool, tr *i18n.Translator) *RegistrationForm {
	f := &RegistrationForm{
		db: db,
		CompanyName: forms.TextInput{
			CommonAttrs:     forms.CommonAttrs{Name: "company_name", ID: "company_name", Required: true},
			Type:            "text",
			Placeholder:     "Acme Telecom Ltd.",
			FieldValidation: forms.FieldValidation{Rules: []validation.Rule{validation.Required(), validation.MinLength(2)}},
		},
		Website: forms.TextInput{
			CommonAttrs: forms.CommonAttrs{Name: "website", ID: "website"},
			Type:        "url",
			Placeholder: "https://acmetelecom.example",
		},
		Email: forms.TextInput{
			CommonAttrs:     forms.CommonAttrs{Name: "email", ID: "email", Required: true},
			Type:            "email",
			Placeholder:     "sales@acmetelecom.example",
			FieldValidation: forms.FieldValidation{Rules: []validation.Rule{validation.Required(), validation.Email()}},
		},
		Phone: forms.NewPhoneInput("phone"), // Tier 2: çevirme kodu SearchableSelect + ulusal numara TextInput
		Country: forms.SearchableSelect{
			BaseSelectField: forms.BaseSelectField{
				CommonAttrs: forms.CommonAttrs{Name: "country", ID: "country", Required: true},
				Placeholder: "Search country…",
				Items:       countryItems(), // []forms.SelectItem
			},
			EventName: "country",
		},
		Tier: forms.Select{
			CommonAttrs: forms.CommonAttrs{Name: "tier", ID: "tier", Required: true},
			Options: []forms.Option{
				{Value: "", Label: "Select a reseller tier"},
				{Value: "silver", Label: "Silver — up to 50 seats/mo"},
				{Value: "gold", Label: "Gold — up to 250 seats/mo"},
				{Value: "platinum", Label: "Platinum — unlimited"},
			},
		},
		MonthlySeats: forms.NumericInput{
			CommonAttrs: forms.CommonAttrs{Name: "monthly_seats", ID: "monthly_seats"},
			Type:        "number",
			Min:         "1",
			Step:        "1",
		},
		AgreeTerms: forms.ChoiceInput{
			CommonAttrs:     forms.CommonAttrs{Name: "agree_terms", ID: "agree_terms", Required: true},
			Type:            "checkbox",
			Value:           "yes",
			LabelText:       "I agree to the 3CX Reseller Program terms",
			FieldValidation: forms.FieldValidation{Rules: []validation.Rule{validation.Required()}},
		},
	}
	for _, field := range []interface{ SetTranslator(*i18n.Translator) }{
		&f.CompanyName, &f.Website, &f.Email, &f.Country, &f.Tier, &f.MonthlySeats, &f.AgreeTerms,
	} {
		field.SetTranslator(tr)
	}
	f.Phone.Number.SetTranslator(tr)
	f.SetTranslator(tr)
	return f
}

func (f *RegistrationForm) Mount(_ context.Context) error   { return nil }
func (f *RegistrationForm) Unmount(_ context.Context) error { return nil }

func (f *RegistrationForm) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
	switch {
	case event == "company_name":
		return f.CompanyName.HandleEvent(ctx, event, payload)
	case event == "website":
		return f.Website.HandleEvent(ctx, event, payload)
	case event == "email":
		return f.Email.HandleEvent(ctx, event, payload)
	case event == "country" || hasPrefix(event, "country."):
		return f.Country.HandleEvent(ctx, event, payload)
	case event == "tier":
		return f.Tier.HandleEvent(ctx, event, payload)
	case event == "monthly_seats":
		return f.MonthlySeats.HandleEvent(ctx, event, payload)
	case event == "agree_terms":
		return f.AgreeTerms.HandleEvent(ctx, event, payload)
	case hasPrefix(event, "phone_dial") || hasPrefix(event, "phone_num"):
		return f.Phone.HandleEvent(ctx, event, payload)
	case event == "register":
		return f.submit(ctx)
	}
	return nil
}

func (f *RegistrationForm) submit(ctx context.Context) error {
	// Çapraz alan kuralı: Platinum katmanı bir aylık koltuk tahmini gerektirir.
	seats, _ := strconv.Atoi(f.MonthlySeats.Value)
	if f.Tier.Value == "platinum" && seats <= 0 {
		f.MonthlySeats.Errors = []string{f.T("reseller.seats_required_for_platinum")}
	} else {
		f.MonthlySeats.Errors = nil
	}

	ok := forms.ValidateAll(
		&f.CompanyName, &f.Email, &f.Country, &f.Tier, &f.AgreeTerms, f.Phone,
	)
	if len(f.MonthlySeats.Errors) > 0 {
		ok = false
	}
	if !ok {
		f.Submitted = false
		f.MarkDirty()
		return nil
	}

	_, err := f.db.Exec(ctx, `
		INSERT INTO reseller_applications
			(company_name, website, email, phone, country, tier, monthly_seats)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, f.CompanyName.Value, f.Website.Value, f.Email.Value, f.Phone.RawValue(),
		f.Country.Value, f.Tier.Value, seats)
	if err != nil {
		f.ToastT("error", "reseller.submit_failed")
		return nil
	}

	f.Submitted = true
	f.Summary = f.CompanyName.Value + " — " + f.Tier.Value + " tier"
	f.ToastT("success", "reseller.submit_success")
	f.MarkDirty()
	return nil
}

func (f *RegistrationForm) Render() (string, error) {
	companyL, _ := (&forms.Label{For: "company_name", Text: f.T("reseller.company_name")}).Render()
	companyI, _ := f.CompanyName.Render()
	webL, _ := (&forms.Label{For: "website", Text: f.T("reseller.website")}).Render()
	webI, _ := f.Website.Render()
	emailL, _ := (&forms.Label{For: "email", Text: f.T("reseller.email")}).Render()
	emailI, _ := f.Email.Render()
	phoneL, _ := (&forms.Label{For: "phone", Text: f.T("reseller.phone")}).Render()
	phoneI, _ := f.Phone.Render()
	countryL, _ := (&forms.Label{For: "country", Text: f.T("reseller.country")}).Render()
	countryI, _ := f.Country.Render()
	tierL, _ := (&forms.Label{For: "tier", Text: f.T("reseller.tier")}).Render()
	tierI, _ := f.Tier.Render()
	seatsL, _ := (&forms.Label{For: "monthly_seats", Text: f.T("reseller.monthly_seats")}).Render()
	seatsI, _ := f.MonthlySeats.Render()
	agreeI, _ := f.AgreeTerms.Render()
	btn, _ := (&forms.Button{Type: "button", Text: f.T("reseller.submit"), EventName: "register"}).Render()

	result := ""
	if f.Submitted {
		o, _ := (&forms.Output{CommonAttrs: forms.CommonAttrs{Name: "summary"}, Text: f.Summary}).Render()
		result = `<div class="result">` + o + `</div>`
	}

	inner := forms.JoinHTML(
		`<div class="field">`, companyL, companyI, `</div>`,
		`<div class="field">`, webL, webI, `</div>`,
		`<div class="field">`, emailL, emailI, `</div>`,
		`<div class="field">`, phoneL, phoneI, `</div>`,
		`<div class="field">`, countryL, countryI, `</div>`,
		`<div class="field">`, tierL, tierI, `</div>`,
		`<div class="field">`, seatsL, seatsI, `</div>`,
		`<div class="field choice">`, agreeI, `</div>`,
		`<div class="actions">`, btn, `</div>`,
		result,
	)
	return (&forms.Form{Method: "post", OnSubmit: "register", InnerHTML: inner}).Render()
}

func hasPrefix(s, prefix string) bool { return len(s) >= len(prefix) && s[:len(prefix)] == prefix }

func countryItems() []forms.SelectItem {
	return []forms.SelectItem{
		{Value: "tr", Label: "Türkiye"},
		{Value: "de", Label: "Germany"},
		{Value: "us", Label: "United States"},
		{Value: "gb", Label: "United Kingdom"},
		// ...
	}
}
```

Onu kaydetmek ve mount etmek, diğer herhangi bir bileşenle aynıdır:

```go
registry.Register("reseller-registration", func() core.Component {
	return reseller.NewRegistrationForm(db, tr)
})
```

Kompozisyon üzerine notlar:

- **Tier 1 alanları** (`TextInput`, `Select`, `ChoiceInput`,
  `NumericInput`), her biri kendi `FieldValidation`'ının sahibidir ve
  kendi WS olayını kendi `HandleEvent`'ine yönlendirir; üst form sadece
  olay adına göre yönlendirme yapar.
- **`forms.PhoneInput`**, yeni bir kontrol ailesi değil, bir Tier 2
  *kompozisyon yardımcısıdır* — dahili olarak bir `SearchableSelect`'e
  (çevirme kodu) ve bir `TextInput`'e (ulusal numara) sahiptir ve tek
  bir `RawValue()` (`"+90 5xx..."`) ile her iki çocuğu doğrulayan tek
  bir `Validate()`'i açığa çıkarır. İki dahili olayı önekli
  (`phone_dial`, `phone_num`), bu yüzden üst formun `HandleEvent`'i tam
  eşleşme yerine önekle dispatch eder.
- **Çapraz alan doğrulaması** (Platinum katmanının bir koltuk sayısı
  gerektirmesi), `forms.ValidateAll`'un üzerine katmanlanmış, `submit`
  içinde sadece normal Go mantığıdır — `ValidateAll`, "her bireysel
  alan kendi kurallarını geçti mi" yarısını halleder; birden fazla
  alana yayılan her şey, `contact-form`'un mevcut alan başına
  kontrolleri gibi, tamamen kendi bileşeninizin sorumluluğudur.
- **Gerçek `INSERT`**, sadece `ok` doğrulandığında çalışır, ve SQL
  parametreleri için sadece doğrulanmış alan değerlerini okur — asla
  ham, doğrulanmamış olay payload verisini değil.

---

## 6. Üretim dağıtımı — Docker'sız

Aşağıdaki her şey, `go build` ile inşa edilmiş tek bir statik Linux
binary'sini, konteyner olmadan, systemd tarafından denetlenen, TLS
sonlandıran bir ters proxy olarak Nginx'in arkasında çalışırken
varsayar.

### 6.1 Build

```bash
GOOS=linux GOARCH=amd64 go build -o /opt/myapp/bin/myapp ./cmd/myapp
```

Derlenmiş binary'i, artı uygulamanızın sunduğu herhangi bir statik
varlığı (`client/`, `forms/`, kendi şablonlarınız/locale'leriniz), hedef
host'a kopyalayın, örn. `/opt/myapp/` altına.

### 6.2 systemd birimi

```ini
# /etc/systemd/system/myapp.service
[Unit]
Description=MyApp (GoUI tabanlı HTTP uygulaması)
After=network.target

[Service]
Type=simple
User=myapp
Group=myapp
WorkingDirectory=/opt/myapp
ExecStart=/opt/myapp/bin/myapp
Restart=on-failure
RestartSec=2
Environment=PORT=8080
Environment=APP_ENV=production
# Uygulamanız bir DATABASE_URL, sırlar vb. okuyorsa, bunları bir
# EnvironmentFile içine koyun:
# EnvironmentFile=/opt/myapp/.env
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now myapp
sudo systemctl status myapp
```

`LimitNOFILE`'ı açıkça yükseltmeye değer: açık her WebSocket bağlantısı,
process'in yaşam döngüsü boyunca bir dosya tanımlayıcısı (file
descriptor) tutar, ve birçok dağıtımdaki varsayılan process başına
sınır (1024), canlı bir sekme açık olan birkaç yüz eşzamanlı kullanıcınız
olduğunda kolayca aşılır.

### 6.3 Nginx ters proxy — WebSocket'e özgü kısımlar

Herhangi bir WebSocket uygulamasını Nginx'in arkasında dağıtırken
insanları tökezleten şey: **`Upgrade`/`Connection` başlıkları varsayılan
olarak proxy'lenmez** ve açıkça yönlendirilmelidir, yoksa `/goui/ws`'teki
WS el sıkışması başarısız olur (tipik olarak istemci tarafında `open`'a
asla ulaşmayan bir bağlantı, veya anlık bir kapanış olarak görünür —
bkz. [14-troubleshooting.md](14-troubleshooting.md) §1).

```nginx
# /etc/nginx/sites-available/myapp.conf

# Genellikle http{} bloğunda veya paylaşılan bir snippet'te bir kez
# gereklidir, böylece aşağıdaki $connection_upgrade map'i aynı server
# bloğunda hem Upgrade hem de Upgrade-olmayan istekler için doğru
# Connection başlığını üretebilir.
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}

server {
    listen 80;
    server_name myapp.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name myapp.example.com;

    ssl_certificate     /etc/letsencrypt/live/myapp.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/myapp.example.com/privkey.pem;

    # Uygulamanızdaki diğer her şey (REST API'leri, sunucu tarafında
    # render edilen sayfalar, statik varlıklar) — özel işlem gerekmeyen
    # düz bir ters proxy.
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # GoUI WebSocket uç noktasının özellikle Upgrade dansına ihtiyacı vardır.
    location /goui/ws {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket'ler uzun ömürlüdür; varsayılan proxy zaman aşımları
        # (genellikle 60s) boştaki bağlantıları sessizce öldürür. Bunları,
        # beklenen boşta kalma sürenizin (veya §7'deki grace period'ın,
        # hangisi daha büyükse) çok üzerine yükseltin, böylece Nginx'in
        # kendisi oturumların bağlantısını kesmez.
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
}
```

Bu ayarla, tarayıcı `wss://myapp.example.com/goui/ws`'ye bağlanır — TLS
Nginx'te sonlanır, ve Nginx ile Go process'iniz arasındaki trafik düz
`ws://127.0.0.1:8080/goui/ws`'dir. `client/goui.js`, şemayı kendisi
türetir:

```js
// client/goui.js
const base = this.wsUrl.startsWith('ws')
  ? this.wsUrl
  : `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}${this.wsUrl}`;
```

dolayısıyla, bootstrap script'iniz `GoUIClient`'i (bu repodaki her
örnekte olduğu gibi) sabit kodlanmış bir `ws://` URL'si yerine
kendi-kaynaklı (same-origin) relatif bir yolla (`'/goui/ws'`) inşa
ettiği sürece, sayfayı `https://` üzerinden sunmak, herhangi bir ek
istemci tarafı yapılandırma olmadan soketi otomatik olarak `wss://`'ye
yükseltir.

### 6.4 TLS sertifikaları

Herhangi bir standart, Docker'sız TLS kurulumu çalışır — en basiti,
Certbot'un Nginx eklentisi:

```bash
sudo apt-get install -y certbot python3-certbot-nginx
sudo certbot --nginx -d myapp.example.com
```

Certbot, `server { listen 443 ssl ... }` bloğunu sizin için yeniden
yazar ve otomatik yenilemeyi ayarlar; yukarıdaki WS'e özgü
`location /goui/ws { ... }` bloğu bu süreçten dokunulmamış kalır —
sadece Certbot'un TLS için yapılandırdığı aynı `server {}` bloğunda
var olduğundan emin olun.

---

## 7. Üretimde performans

Üç kaldıraç, hepsi bu belgelerde başka yerlerde ayrıntılı olarak
kapsanmıştır — gerçek bir dağıtım için bir kontrol listesi olarak
özetlenmiştir:

- **Birincil navigasyonu prefetch edin.** Kullanıcıların sonra
  açması muhtemel görünümlere (sekmeler, sihirbaz adımları, bir
  listeden "detayları görüntüle") giden linklere `data-goui-prefetch`/
  `data-goui-activate` koyun, böylece kullanıcı tıkladığında `Mount`
  zaten çalışmış olur — bkz. [09-prefetch.md](09-prefetch.md). `Mount`'unun
  yan etkileri olan herhangi bir şeyi prefetch **etmeyin**, ve oturum
  başına sınırın LRU tahliyesi ile zorlanan `ws.MaxPrefetch` (5)
  olduğunu unutmayın — prefetch, tüm navigasyon menünüzü ısıtmanın bir
  yolu değil, bir avuç "muhtemelen sıradaki" hedef içindir.

- **Büyük/yeniden sıralanabilir listelerde `data-key`.** Filtrelenebilecek,
  sıralanabilecek, veya ortadan eklenip çıkarılabilecek (sadece eklenen
  değil) bir avuçtan fazla satırla desteklenen herhangi bir liste, her
  satırda stabil bir `data-key` taşımalıdır — aksi halde böyle her
  değişiklik, ilk farklı indeksten itibaren her şeyi değiştirmeye
  dejenere olur. Key'li diff'lemenin tam olarak nasıl davrandığı ve
  hâlâ keskin kenarları olan yerler için (ağır eşzamanlı çok-öğeli
  yeniden sıralamalar) [10-diffing-internals.md](10-diffing-internals.md)
  §5 ve §7'ye bakın.

- **Yeniden bağlanma grace period'ını `ws.NewHubWithGracePeriod` ile
  ayarlayın.** Varsayılan olarak, `ws.NewHub()`, `ws.DefaultGracePeriod`'u
  (60 saniye) kullanır — bağlantısı kesilmiş bir oturum (dizüstü
  bilgisayarın kapağı kapandı, kısa bir ağ kesintisi, mobil sekme arka
  plana alındı), mount edilmiş bileşenlerini canlı ve giden kuyruğunu
  o kadar süre tamponlayarak tutar, dolayısıyla bir yeniden bağlanma,
  tam bir yeniden mount değil, hızlı, şeffaf bir devam etmedir.
  Üretimde genellikle bunu varsayılanı sessizce kabul etmek yerine
  açıkça düşünmek istersiniz:

  ```go
  // Sekmeleri sık sık arka plana alan mobil-ağırlıklı bir kullanıcı
  // tabanı için daha uzun bir grace period, düşüşten sonra bileşen
  // başına daha uzun süre mount edilmiş durumu (ve DB bağlantılarını,
  // goroutine'leri vb.) tutma bedeliyle.
  hub := ws.NewHubWithGracePeriod(3 * time.Minute)
  server := ws.NewServer(hub, registry, tr)
  gouifiber.Register(app, gouifiber.Options{Server: server})
  ```

  Kasıtlı olarak yapılacak trade-off: **daha uzun** bir grace period,
  daha yumuşak yeniden bağlanmalar (durum ve prefetch kısa
  kesintilere hayatta kalır) ama süresi dolmamış, bağlantısı kesik
  oturum başına daha fazla sunucu tarafı kaynak tutulması anlamına
  gelir; **daha kısa** bir tane kaynakları daha hızlı serbest bırakır
  ama kısa ağ kesintilerini tam yeniden mount'lara (taze `Mount`, taze
  ilk render, kaybolmuş herhangi bir prefetch ilerlemesi) çevirir.
  Hangi değeri seçerseniz seçin, Nginx'in `proxy_read_timeout`/
  `proxy_send_timeout`'unu (§6.3) da onun rahatlıkla üzerine
  yükseltin — önündeki ters proxy boşta bağlantıları daha erken
  düşürüyorsa, cömert bir uygulama seviyesi grace period'ın hiçbir
  faydası yoktur.
</contents>
