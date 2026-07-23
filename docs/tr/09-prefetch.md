# 09. Prefetch

Prefetch, tarayıcının bir bileşenin `Mount(ctx)`'ini kullanıcı ona
gerçekten gitmeden *önce* çalıştırmasını sunucudan istemesine olanak
tanır, böylece kullanıcı gittiğinde, ilk render bir gidiş-dönüş artı
`Mount`'un yaptığı her ne işse (veritabanı okumaları, önbellek ısıtma
vb.) beklemek yerine anında olur.

Bu belge boyunca kullanılan modül yolu: `github.com/zatrano/goui`.

## 1. İki öznitelik

Prefetch tamamen özniteliğe (attribute) dayalıdır; çoğu uygulamada
doğrudan herhangi bir JS API'sini çağırmazsınız.

```html
<a href="/contact" data-goui-prefetch="contact" data-goui-activate="contact">
  Go to contact form
</a>
```

- **`data-goui-prefetch="<registry-name>"`** — bir elemanı,
  `<registry-name>` altında kayıtlı bileşen için bir prefetch tetikleyicisi
  olarak işaretler. İstemci modülü `client/modules/prefetch.js` onu şu
  zaman prefetch eder:
  - imleç elemanın üzerinde **~100ms** durduğunda (`HOVER_MS`), veya
  - eleman `IntersectionObserver` ile, `80px`'lik bir root margin ile
    viewport'a kaydırıldığında — yani, gerçekten görünür olmadan biraz
    önce yüklemeye başlar.

  Her isim istemci oturumu başına sadece bir kez istenir
  (`prefetch.js` bir `requested` `Set`'i takip eder), dolayısıyla aynı
  linkin üzerine yeniden gelmek veya onun yanından yeniden kaydırmak
  istemci tarafında no-op'tur.

- **`data-goui-activate="<registry-name>"`** — tıklamada, varsayılan
  navigasyonu önler, hemen bir prefetch talep eder (henüz talep
  edilmediyse) ve o isim için bir **activate** frame'i gönderir. Prefetch
  edilen bileşeni render edilmiş, görünür bir bileşene yükselten
  öznitelik budur.

`data-goui-prefetch`'i tek başına kullanabilirsiniz (sadece onu ısıtın,
sonra başka bir mekanizma/olayla activate edin), veya yukarıda gösterildiği
gibi ikisini de aynı elemanda birleştirebilirsiniz — `examples/contact-form`'un
`Landing` bileşeninin kullandığı yaygın "hover'da ısıt, tıklamada takas et"
deseni:

```go
func (l *Landing) Render() (string, error) {
	return `<div class="landing">
  <p><a href="#" data-goui-prefetch="contact" data-goui-activate="contact">Go to the contact form</a></p>
</div>`, nil
}
```

## 2. Tel protokolü

`ws/frame.go`'da tanımlanan iki yeni frame türü:

```go
const (
	// ...
	FrameTypePrefetch = "prefetch"
	FrameTypeActivate = "activate"
)
```

İstemci (`client/goui.js`), bunları sadece bir bileşen *tür ismi* taşıyan
düz frame'ler olarak gönderir (henüz bir örnek ID'si değil, çünkü
bileşen `Prefetch`/`Activate` çalışana kadar sunucu tarafında yoktur):

```js
sendPrefetch(componentName) {
  this.ws.send(JSON.stringify({ type: 'prefetch', component: componentName }));
}

sendActivate(componentName) {
  this.ws.send(JSON.stringify({ type: 'activate', component: componentName }));
}
```

Sunucuda, `Session.readLoop` (`ws/session.go`) bunları sırasıyla
`Session.Prefetch` ve `Session.Activate`'e dispatch eder.

## 3. Sunucuda ne olur

### 3.1 `Session.Prefetch(name)`

```go
func (s *Session) Prefetch(name string) error {
	// name boşsa, veya zaten prefetch edildiyse no-op
	// registry.Create(name) → prepareComponent (SetTranslator, SetPusher, Mount(ctx))
	// s.prefetched[name]'e ekle ve s.prefetchOrder'a ekle
	// len(s.prefetchOrder) >= MaxPrefetch ise en eskisini tahliye et
}
```

Anahtar özellikler:

- Oturumun `*core.Registry`'si aracılığıyla gerçek bir bileşen örneği
  **oluşturur ve `Mount` eder**, çevirmen ve toast pusher'ı normalde
  activate edilmiş bir bileşende olacağı gibi zaten bağlanmıştır.
- **Render etmez** ve istemciye **hiçbir frame göndermez**. Bir bileşeni
  prefetch etmek, prefetch talebinin kendisinin ötesinde sıfır WebSocket
  trafiği üretir — test paketi bunu doğrular
  (`TestSession_Prefetch_MountsWithoutRender`, bir `Prefetch` çağrısından
  sonra hiçbir giden frame olmadığını doğrular).
- Aynı isim için yinelenen prefetch'ler no-op'tur: `name` zaten
  `s.prefetched`'de ise, `Prefetch` ikinci bir örnek oluşturmadan veya
  `Mount`'u yeniden çağırmadan hemen döner.
- Örnek, `s.prefetched[name]`'de yaşar; bu, üretilmiş bir bileşen ID'sine
  göre değil, **registry ismine** göre anahtarlanmış bir map'tir —
  kasıtlı olarak henüz sunucu tarafından atanmış bir ID yoktur, çünkü
  bileşen activate edilene kadar "canlı" değildir.

### 3.2 `Session.Activate(name)`

```go
func (s *Session) Activate(name string) (string, error) {
	// name, s.prefetched'te ise: o örneği yeniden kullan (prefetched map'ten sil)
	// aksi halde: registry.Create(name) taze, sonra Mount et
	// taze bir bileşen ID'si ata, s.components[id]'de sakla
	// sendFullRender(id, component)
	return id, nil
}
```

- İsim daha önce prefetch edildiyse, `Activate` **tam olarak aynı örneği
  yeniden kullanır** — ikinci bir `Mount` çağrısı yok, ve `Mount`
  sırasında biriken herhangi bir durum (veya activate edilmeden önce
  bile daha sonra mutasyona uğrayan, gerçi hiçbir şey prefetch edilmiş
  bir bileşeni mutasyona uğratmamalıdır) korunur. Bu,
  `TestSession_Prefetch_ActivateUsesExisting` ile doğrulanır.
- İsim prefetch edilmediyse (örn. kullanıcı 100ms hover zamanlayıcısı
  veya intersection observer ateşlenmeden önce tıkladıysa, veya
  `data-goui-activate`, `data-goui-prefetch` olmadan kullanıldıysa),
  `Activate`, olduğu yerde taze bir örnek oluşturmaya ve mount etmeye
  şeffaf bir şekilde geri döner — çağıranın bakış açısından, prefetch
  yalnızca bir optimizasyondur; doğruluk ona bağlı değildir.
- Her durumda, activation, bileşenin gerçek bir bileşen ID'si aldığı ve
  **ilk tam render'ının** gönderildiği noktadır
  (`Op: diff.OpReplace, Path: []`), tam olarak yeni mount edilmiş başka
  herhangi bir bileşenin ilk render'ı gibi.

## 4. `MaxPrefetch` ve LRU tahliyesi

```go
// ws/frame.go
// MaxPrefetch, sessizce mount edilmiş (henüz görünür olmayan) bileşenler için oturum başına sınırdır.
const MaxPrefetch = 5
```

Her oturum, aynı anda en fazla 5 prefetch edilmiş-ama-henüz-activate-edilmemiş
bileşen tutabilir. `s.prefetchOrder`, ekleme sırasını takip eder (en eski
önce); farklı bir 6. prefetch geldiğinde, en eski girdi tahliye edilir:

```go
for len(s.prefetchOrder) >= MaxPrefetch {
	oldest := s.prefetchOrder[0]
	s.prefetchOrder = s.prefetchOrder[1:]
	if old, ok := s.prefetched[oldest]; ok {
		delete(s.prefetched, oldest)
		evicted = append(evicted, old)
	}
}
```

Tahliye edilen bileşenler düzgünce `Unmount(ctx)` edilir (oturum kilidi
dışında, tahliye defter tutumundan sonra) — bu bir sızıntı değildir,
gerçek bir söküm işlemidir, dolayısıyla `Mount`/`Unmount`, bir bileşen
hiç activate edilmese bile eşleşen bir çift olarak yazılmalıdır. Bu üst
sınır, tek bir tarayıcı sekmesinin sunucuya zorlayabileceği spekülatif
işin miktarını sınırlamak için vardır (örn. bir kullanıcının art arda
on navigasyon linkinin üzerine gelmesi, sunucuda sonsuza kadar on canlı
DB bağlantısı sıcak bırakmamalıdır).

Hiç activate edilmeyen prefetch edilmiş bileşenler, oturumun kendisi
WebSocket grace period'ını geçtiğinde de temizlenir — bkz.
[13-project-integration.md](13-project-integration.md) §7 ve
`TestSession_Prefetch_CleanedOnGracePeriodExpiry`.

## 5. Prefetch ne zaman kullanılır

Prefetch, şu durumlarda uygun bir seçimdir:

- `Mount`, kullanıcı tıkladığında zaten tamamlanmış olmasını
  isteyeceğiniz gerçek, muhtemelen yavaş bir iş yapıyor (bir DB
  sorgusu, harici bir API çağrısı, bir önbellek araması).
- Hedef *muhtemelen* ziyaret edilecek — birincil navigasyon, bir
  sihirbazdaki (wizard) "sonraki adım" linkleri, sekmeler, bir hover
  kartından açılan bir detay görünümü.
- `Mount` **idempotent ve yan etkisizdir** — onu çağırıp sonra sonucu
  (bir tahliye veya bağlantısı kesilen bir oturum aracılığıyla) kötü
  bir şey olmadan basitçe atmak sorun değil.

## 6. Prefetch *ne zaman* kullanılmamalı

- **`Mount`'un yan etkileri var.** Bir bileşeni mount etmek bir e-posta
  gönderiyorsa, bir denetim günlüğü satırı ekliyorsa, bir sayaç
  artırıyorsa, özel bir kaynak edinilyorsa, veya "sadece fare bir
  linkin üzerine geldi diye" olmaması gereken başka bir şey yapıyorsa,
  onu prefetch etmeyin. Unutmayın: prefetch edilmiş bir bileşenin
  `Mount`'u, kullanıcı asla tıklamasa da, sunucuda gerçekten tam olarak
  çalışır.
- **Nadiren ziyaret edilen hedefler.** Kimsenin tıklamadığı bir linki
  prefetch etmek bir `Mount`/`Unmount` döngüsünü israf eder ve 5
  prefetch slotundan birini işgal eder (gerçekten yararlı olabilecek
  bir şeyi tahliye ederek).
- **Zaten ucuz olan `Mount`.** Mount etmek sadece bir struct alanını
  sıfırlamaksa, prefetch, ölçülebilir bir fayda olmadan bir ağ
  gidiş-dönüşü ve defter tutumu ekler — sadece doğrudan activate edin.
- **Yüksek derecede dinamik bileşen isimleri.** Prefetch, bileşenleri
  yalnızca registry ismine göre tanımlar. Göstermek istediğiniz gerçek
  bileşen o isimde yakalanmayan istek zamanı parametrelerine bağlıysa
  (bir satır ID'si, bir arama sorgusu), genel ismi prefetch etmek yanlış
  şeyi ısıtabilir, veya hiç yararlı bir şeyi. Prefetch, sabit bir
  isimli hedefler kümesine (sekmeler, sihirbaz adımları, yaygın
  sayfalar) en uygundur, kayıt başına detay görünümlerine değil (kayıt
  başına bir isim kaydetmediğiniz sürece, ki bu genellikle pratik
  değildir).

## 7. Ön-render yok — bu neden önemli

Aksini varsaymak kolay olduğundan tekrar etmeye değer: prefetch **asla
render etmez** ve tarayıcıya **asla HTML göndermez**. Size sağladığı
tek şey, activation zamanında, `Mount`'un zaten gerçekleşmiş olmasıdır.
Activation çağrısı hâlâ şunları yapar:

1. Bir bileşen ID'si atar.
2. `Render()`'ı çağırır.
3. HTML'i `data-goui-component="<id>"` ile sarar/dekore eder.
4. Onu bir `diff.Node` ağacına ayrıştırır.
5. Onu kök yolda tek bir `OpReplace` yaması olarak gönderir.

Yani prefetch, `Mount` gecikmesini kritik yoldan tıraşlar, ama
render/serileştir/ayrıştır/gönder adımları hâlâ activation zamanında
eşzamanlı olarak gerçekleşir, diğer herhangi bir ilk render gibi.
Darboğazınız yavaş bir `Mount()` yerine yavaş bir `Render()`'sa, prefetch
yardımcı olmaz — `Render()`'ı optimize edin, veya onun hesaplaması
gerekeni azaltın.
</contents>
