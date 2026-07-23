# 14. Sorun Giderme

GoUI üzerine inşa ederken en sık ortaya çıkan başarısızlık modları için
kontrol-listesi tarzı bir referans. Her bölüm, bir olay sırasında
gözden geçirilebilir olacak şekilde tasarlanmıştır — gördüğünüz belirtiye
atlayın.

Bu belge boyunca kullanılan modül yolu: `github.com/zatrano/goui`.

---

## 1. WebSocket bağlanmıyor

Belirtiler: tarayıcı hiçbir zaman bir `"session"` frame'i almıyor,
`onConnect` hiç ateşlenmiyor, `onError` hemen ateşleniyor, veya soket
açıldıktan hemen sonra kapanıyor.

Bunları sırayla inceleyin:

- [ ] **`/goui/ws` gerçekten erişilebilir mi?** Adapter'ınız
  `GET /goui/ws` (`ws.Path`) yolunu bağlar ve WebSocket yükseltmesi olmayan
  istekleri reddeder (Fiber `fiber.ErrUpgradeRequired` döndürür;
  stdlib/Gin/Echo kendi yükseltme kontrollerini kullanır — bkz. `adapters/`).
  URL'yi düz bir tarayıcı navigasyonu/`fetch` ile vurmak doğru bir şekilde
  başarısız olur — bu bir hata değildir. *İstemcinin* gerçekten bir
  `ws://`/`wss://` URL'si inşa ettiğini, `http://`/`https://` değil,
  onaylayın.
- [ ] **URL kendi-kaynaklı/relatif mi, yoksa sayfanın sunulma şekliyle
  eşleşmeyen bir şema/host'u yanlışlıkla sabit kodladınız mı?**
  `GoUIClient._buildUrl`, yapılandırılmış `wsUrl`, `ws` ile
  **başlamadığında** sadece `ws:`/`wss:`'i otomatik olarak türetir —
  yanlışlıkla tam nitelikli bir `http://` URL'si geçirirseniz, sizin
  için düzeltilmez.
- [ ] **Bir ters proxy'nin arkasında mı?** Nginx (ve çoğu proxy),
  açıkça `proxy_http_version 1.1;`, `proxy_set_header Upgrade
  $http_upgrade;`, ve `proxy_set_header Connection $connection_upgrade;`'i
  yapılandırmadığınız sürece `Upgrade`/`Connection: Upgrade`
  başlıklarını **yönlendirmez**. Bunu kaçırmak, "yerelde çalışıyor ama
  üretimde çalışmıyor"un en yaygın dağıtım zamanı nedenidir. Tam,
  çalışan bir Nginx yapılandırması için
  [13-project-integration.md](13-project-integration.md) §6.3'e bakın.
- [ ] **`?session=` olmayan taze bir bağlantıda `?component=` atlandı
  mı?** Bilinen bir `session` ne de bir `component` sorgu parametresi
  sağlanmadıysa, sunucu bir hata frame'i (`ws.ErrComponentRequired`,
  `"component query parameter is required"`) ile yanıt verir ve
  bağlantıyı kapatır. `GoUIClient`'in boş olmayan bir `componentName`
  ile inşa edildiğini onaylayın.
- [ ] **Bileşen ismi gerçekten kayıtlı mı?** `registry.Create`, bilinmeyen
  bir isim için `core.ErrComponentNotRegistered` döndürür, ki bunu
  sunucu bir hata frame'ine çevirir ve ardından bir `"session"`
  frame'i hiç göndermeden bağlantıyı kapatır. `registry.Register(...)`'a
  geçirilen tam string'in istemcinin istediğiyle eşleştiğini iki kez
  kontrol edin — bu, [13-project-integration.md](13-project-integration.md)
  §2'de açıklanan kiracı-nitelikli isimlerle özellikle yanlış anlaşılması
  kolaydır.
- [ ] **Tarayıcının Network/WS incelemecisinde gerçek kapanış kodunu ve
  yükseltmeden önceki herhangi bir HTTP yanıtını kontrol edin.**
  Hiç gelmeyen bir `101 Switching Protocols` (`pending`'de takılı
  kalmış, veya `4xx`/`5xx` yerine) proxy/routing katmanına işaret eder;
  açılıp sonra hemen kapanan bir soket, yukarıdaki hata-frame'i yoluna
  işaret eder — kapanmadan önce, herhangi biri geldiyse, `"error"`
  frame'inin payload'ını okuyun.
- [ ] **Sunucu tarafı günlükler.** `Session.readLoop`, prefetch
  başarısızlıklarını günlükler (`log.Printf("[goui] prefetch %q failed: %v", ...)`)
  ama çoğu bağlantı seviyesi başarısızlık, sadece istemciye gönderilen
  hata frame'i olarak görünür — varsayılan olarak ayrı bir sunucu
  tarafı WS bağlantı günlüğü yoktur. Tekrarlanan başarısızlıklara
  sunucu tarafı görünürlük gerekiyorsa, `registry.Create`/`hub.Get`
  çağrılarının etrafına kendi günlüklemenizi ekleyin.

---

## 2. Bayat oturum (`sessionStorage` key'i `goui.sessionId`)

İstemci, oturum ID'sini `sessionStorage`'da `goui.sessionId` key'i
altında kalıcı hale getirir (`client/goui.js`'de `SESSION_KEY`), böylece
sadece bir ağ kesintisi değil, bir sayfa yeniden yükleme, baştan
başlamak yerine *aynı* sunucu tarafı `Session`'a (ve dolayısıyla aynı
mount edilmiş bileşen durumuna) yeniden bağlanabilir.

**Belirti:** bir sunucu yeniden başlatmasından (bkz. §6) veya bir `Hub`
yeniden oluşturmasından sonra, tarayıcı hâlâ `sessionStorage`'da eski
bir `goui.sessionId`'ye sahiptir, `?session=<old-id>` ile yeniden
bağlanmayı dener, ve sunucu — o ID hakkında artık hiçbir hafızası
olmayan — `"session not found"` ile yanıt verir ve yeni bir oturum
kaydetmeden bağlantıyı kapatır.

İstemcinin tam olarak bu durum için kendi kendini iyileştiren bir yolu
zaten vardır — kurulumunuzda gerçekten erişilebilir olduğunu doğrulayın:

```js
// client/goui.js — _handleFrame, case 'error'
if (message === 'session not found' || message.includes('session not found')) {
  this.sessionId = '';
  sessionStorage.removeItem(SESSION_KEY);
  this.componentRoots.clear();
  if (this.ws) this.ws.close();
  this.onError(message + ' — reconnecting fresh');
  return;
}
```

Bu otomatik olarak *kurtarılmıyorsa* kontrol listesi:

- [ ] **`onclose`, gerçekten `_scheduleReconnect`'e bağlı mı?**
  `GoUIClient`'in yerleşik yeniden bağlanma mantığını geçersiz kıldıysanız
  veya atladıysanız (özel bir taşıma sarmalayıcısı, elle `WebSocket`
  kullanımı vb.), yukarıdaki otomatik "bayat ID'yi temizle ve taze
  yeniden bağlan" yolu asla çalışmaz — bunu yeniden üretmeniz gerekir.
- [ ] **Eski oturum ID'sinin bir kopyasını tutan başka bir şey mi var**
  (yükleme zamanında onu gömen sunucu tarafında render edilmiş bir
  sayfa, bir çerez, özel bir depolama) ve istemci `sessionStorage`'ı
  zaten temizledikten sonra onu yeniden mi enjekte ediyor? Yeniden
  bağlanmada `?session=` için *tek* doğruluk kaynağının,
  `_buildUrl()` zamanında taze okunan
  `sessionStorage.getItem('goui.sessionId')` olduğunu onaylayın.
- [ ] **"Çıkış yap" / "kiracı değiştir" akışı için elle durumu mu
  temizliyorsunuz?** Farklı bir bileşen/kiracı ismiyle yeniden
  bağlanmadan önce `sessionStorage.removeItem('goui.sessionId')`'yi
  (veya `sessionStorage.clear()`'ı) kendiniz çağırın — aksi halde
  tarayıcı, (doğru bir şekilde) istediğiniz yeni bağlamla hiçbir
  ilişkisi olmayan bir oturuma yeniden bağlanmayı deneyecektir.
- [ ] **Birden fazla sekme beklenmedik şekilde bir oturum ID'sini mi
  paylaşıyor?** `sessionStorage`, spesifikasyona göre sekme başınadır,
  ama ID'yi kendi kodunuzda bir yerde `localStorage`'a veya bir çereze
  kopyaladıysanız, iki sekme aynı sunucu `Session`'ına yeniden
  bağlanmak için yarışabilir, ve yarışı kaybedene
  `ws.ErrSessionAlreadyActive` döndürülür (`Session.Reattach`, bir
  bağlantı zaten aktifse yeniden bağlanmayı reddeder). Bir oturumu
  sekmeler arasında paylaşmayı özellikle amaçlamadığınız sürece
  (GoUI bunu kutudan çıktığı gibi desteklemez), oturum ID'lerini
  `sessionStorage`'a kapsamlı tutun.

---

## 3. Yama yolu "bir kayma" gibi görünüyor / yamalar yanlış elemana iniyor

Belirti: bir `set_attr`/`update_text`/`replace` yaması bir kardeşi
hedefliyor gibi görünüyor, görünürde hiçbir şey olmuyor, veya eksik bir
hedef hakkında bir hata görünüyor.

- [ ] **Yolların, ham DOM `childNodes`'a değil, "anlamlı çocuklar"
  aracılığıyla çözülen, bileşen köküne göre relatif olduğunu
  unutmayın.** Hem sunucu (`diff.ParseHTML`/`convertHTMLNode`) hem de
  istemci (`client/goui.js`'deki `meaningfulChildren()`) çocukları
  indekslemeden önce sadece-boşluk metin düğümlerini düşürür.
  İndeksleri ham tarayıcı DevTools `childNodes` çıktısına karşı
  karşılaştırıyorsanız (HTML string'inizdeki girintiden gelen boşluk
  metin düğümlerini *dahil eder*), indeksler eşleşmez — her zaman
  indeksleri *elemanlar + boş-olmayan metin* açısından düşünün, o
  `meaningfulChildren`'ın hesapladığıyla eşleşecek şekilde.
- [ ] **`Render()`, bir kök elemandan fazlasını mı döndürdü?** Öyleyse,
  GoUI çıktınızı sentetik bir `<div data-goui-component="...">` içine
  sarar (bkz. [10-diffing-internals.md](10-diffing-internals.md) §3),
  ki bu, kendi `Render()` çıktınızı yalıtılmış olarak okurken
  beklediğinizden bir seviye daha aşağı her yolu kaydırır. `Render()`'ın
  tam olarak bir kök eleman döndürmesini sağlayarak düzeltin — bu, hem
  ekstra iç içe geçmeyi kaldırır hem de istemcinin
  `[data-goui-component]` ile aradığı elemanla eşleşir.
- [ ] **Diff'lenen önceki render ağacı, gerçekten diff'lenen ağaç mı?**
  `Session.renderTrees[componentID]`, sadece başarılı bir `Render()` +
  ayrıştırmadan sonra `sendRender`/`sendFullRender` tarafından
  ayarlanır. Önceki bir `Render()` çağrısı kendi istek işlemenizin
  ortasında hata verdiyse (ve hatayı yuttunuzsa), saklanan ağaç, o an
  canlı DOM'da olan şeyden daha eski olabilir; bu, *saklanan* ağaca
  karşı doğru görünen ama kullanıcının gerçekten görmesine karşı yanlış
  görünen yamalar üretir. Her `HandleEvent` yolunun ya bir hata
  döndürdüğünden (istemciye bir hata frame'i olarak yansıtılır — bkz.
  `Session.handleEventFrame`) ya da bileşeni, sonraki `Render()`
  çıktısı gerçeklikle eşleşen bir durumda bıraktığından emin olun.
- [ ] **`data-goui-ignore` ile işaretlenmemiş bir bileşen alt ağacı
  içinde GoUI'nin dışında DOM'u elle mi düzenlediniz** (bir tarayıcı
  eklentisi, başka bir script, elle DevTools düzenlemesi)? GoUI'nin
  diff'lemesi, canlı DOM'un render ettiği son ağaçla eşleştiğini
  varsayar; uzlaştırılmış bir alt ağaç içindeki herhangi bir bant
  dışı mutasyon, sonraki yamalar için indeksleri senkron dışına
  çıkarabilir. GoUI'nin onun üzerine uzlaştırma yapmasına izin vermek
  yerine, kasıtlı olarak istemciye ait olan herhangi bir bölgeyi
  `data-goui-ignore` içine sarın (§5).

---

## 4. Bir yamadan sonra Checkbox/radio durumu görsel olarak eşleşmiyor

Belirti: sunucu açıkça `set_attr checked="checked"`'i gönderdi (veya
kaldırdı), ama checkbox/radio'nun tarayıcıdaki görsel durumu eşleşecek
şekilde değişmiyor — veya değişiyor, sonra sonraki ilgisiz bir
etkileşimde geri dönüyor.

Bu iyi bilinen bir DOM tuhaflığıdır, bir GoUI hatası değil, ama GoUI'nin
bunu nasıl ele aldığını tam olarak bilmeye değer, böylece işlemenin
kendisi atlandığında tanıyabilirsiniz: `checked`, `selected`,
`disabled`, ve `readonly` için, bir checkbox ile etkileşime girildikten
sonra **HTML öznitelikleri ve DOM özellikleri iki farklı şeydir** —
`checked` *özniteliğini* `setAttribute` aracılığıyla ayarlamak, canlı
`.checked` *özelliğini*, tarayıcının gerçekten render ettiği şeyi,
güvenilir bir şekilde güncellemez. `client/goui.js`'nin `applyPatch`'i,
bunu hem `set_attr` hem de `remove_attr` için açıkça ele alır:

```js
case 'set_attr': {
  const target = resolvePath(rootEl, path);
  if (target && target.nodeType === Node.ELEMENT_NODE) {
    const name = patch.attr;
    const value = patch.value ?? '';
    target.setAttribute(name, value);
    // Boolean DOM özellikleri, yamalamadan sonra form kontrolleri için senkron tutulmalıdır.
    if (name === 'checked' || name === 'selected' || name === 'disabled' || name === 'readOnly' || name === 'readonly') {
      const prop = name === 'readonly' ? 'readOnly' : name;
      target[prop] = true;
    }
    if (name === 'value' && 'value' in target) {
      target.value = value;
    }
  }
  break;
}
```

Hâlâ bir uyumsuzluk görüyorsanız kontrol listesi:

- [ ] **`data-goui-ignore` ile işaretlenmiş bir alt ağacı mı
  yamalıyorsunuz?** `applyPatch`, hedef (veya insert/remove için,
  ebeveyni) yok sayılmış bir bölgenin içindeyse (§5) yukarıdaki koda
  ulaşmadan önce çıkar — kasıtlı olarak, böylece istemciye ait
  widget'lara dokunulmaz. Checkbox'ınız ilgisiz bir sebeple böyle bir
  bölgenin içindeyse, `checked` durumu GoUI tarafından asla senkronize
  edilmez; tamamen o bölgenin sahibi olan her ne ise ona bağlıdır.
- [ ] **Yama gerçekten tarayıcıya ulaştı mı?** Bu olay için gerçekten
  bir `checked`/`value` `set_attr`/`remove_attr` yamasının gönderildiğini
  ağ incelemecisi (veya geçici bir `onError`/console günlüğü) aracılığıyla
  onaylayın — sunucu tarafı durum mutasyona uğradıktan sonra
  `MarkDirty()` hiç çağrılmadıysa, hiç render (ve dolayısıyla hiç yama)
  üretilmez; bileşenin *sonraki* ilgisiz render'ı sonunda değişikliği
  yansıtır, ki bu "bir sonraki tıklamada kendi kendini düzeltti" gibi
  görünebilir. `forms.ChoiceInput.HandleEvent`, tanınan bir değişiklik
  olayında koşulsuz olarak `MarkDirty()`'yi çağırır — özel bir checkbox
  kontrolü inşa ettiyseniz, sizinkinin de öyle yaptığından emin olun.
- [ ] **Bir `value`/`checked` değeri, o özelliğin uygulanmadığı bir
  eleman türünde mi ayarlanıyor** (örn. bir `<input>` yerine düz bir
  `<div>`'de `checked` ayarlamak)? Yukarıdaki özellik-senkron dalı
  sadece `target[prop]`'u atar — eleman türünün o özelliği desteklediğini
  doğrulamaz; boolean olmayan bir elemana `checked` atamak, DOM'un
  bakış açısından sessizce bir no-op'tur, ki bu "yama uygulanmadı"ya
  aynı görünebilir.

---

## 5. Rich text / kod editörü imleci atlıyor, veya düzenlemeler eziliyor

Belirti: bir Quill (`forms.RichTextEditor`) veya CodeMirror
(`forms.CodeEditor`) örneğinde yazmak imlecin alanın başına/sonuna
atlamasına, seçimlerin kaybolmasına, veya editörün tamamının yazarken
görünür bir şekilde yeniden mount edilmesine sebep oluyor.

Bu, GoUI'nin diff/patch uzlaştırmasının, üçüncü taraf bir editör
kütüphanesinin sahip olduğu ve kendi başına mutasyona uğrattığı DOM'a
dokunmasına izin verildiğinde olur. Düzeltme her zaman aynı üç
mekanizmanın bir kombinasyonudur — kontrolünüz için hepsinin gerçekten
yerinde olduğunu kontrol edin:

- [ ] **Editörün mount elemanında `data-goui-ignore`.** Hem
  `RichTextEditor.Render()` hem de `CodeEditor.Render()`, kendi dış
  sarmalayıcılarında `data-goui-ignore="1"`'i ayarlar. `client/goui.js`'nin
  `isGoUIIgnored`/`applyPatch`'i, çözülmüş hedefi (veya insert/remove
  için, çözülmüş *ebeveyni*) bir `[data-goui-ignore]` atasının içinde
  olan herhangi bir yamayı atlar — bu, GoUI'nin uzlaştırıcısının ilk
  render'dan sonra, sunucu tarafında ne değişirse değişsin, o elemanın
  içindeki hiçbir şeye asla dokunmayacağı anlamına gelir. Özel bir
  üçüncü-taraf-widget sarmalayıcısı yazıyorsanız, bu özniteliği onun
  dış elemanına kopyalayın.
- [ ] **Stabil, "render'da-boş" bir senkronizasyon kanalı, canlı değer
  değil.** Ne `RichTextEditor` ne de `CodeEditor`, mevcut `Value`'yu her
  render'da yok sayılmış alt ağaca render eder — senkronizasyon
  `<textarea class="goui-editor-sync">`'i kasıtlı olarak her zaman boş
  render edilir, *başlangıç* değeri sadece mount zamanında JS köprüsü
  tarafından (bir kez okunan) `data-initial` aracılığıyla mevcuttur —
  bkz. `client/modules/richtext.js`'nin `mountQuill`'i. Özel bir kontrol
  bunun yerine widget'ın *mevcut* metnini her sunucu gidiş-dönüşünde
  markup'a yeniden render ediyorsa, `data-goui-ignore` yerinde olsa
  bile diff yamalayıcısıyla savaşırsınız, çünkü onun yukarısındaki
  şekli sürekli değişen bir `data-goui-ignore`'lu eleman, hâlâ
  *ebeveynin* yeniden anahtarlanmasına/değiştirilmesine sebep olabilir.
- [ ] **Bir render başka türlü gerekli değilse, saf DOM-senkron olayları
  için `HandleEvent`'ten `core.ErrSkipRender` döndürün.**
  `Session.handleEventFrame`, bu sentinel'i açıkça kontrol eder:

  ```go
  if err := component.HandleEvent(ctx, frame.Event, payload); err != nil {
      if errors.Is(err, core.ErrSkipRender) {
          return // olayı onayla, ama hiçbir şeyi render/yama etme
      }
      // ...
  }
  ```

  `RichTextEditor`/`CodeEditor`'ın kendi `HandleEvent` uygulamalarının
  şu anda sadece `Value`'yu güncelleyip `ErrSkipRender`'ı doğrudan
  döndürmek yerine `MarkDirty()`'yi atladığını (üst formun ne zaman
  yeniden render edileceğine karar vermesine güvenerek) unutmayın —
  her iki yaklaşım da, hiçbir görünür fayda sağlamayan bir değer için
  gereksiz bir yama döngüsünden kaçınır. Bir olay tamamen sunucu tarafı
  durumu istemciye ait bir widget'la senkron tutmak için var olduğunda
  ve bir yeniden render hiçbir görünür fayda sağlamayacaksa (ve yok
  sayılan bölgenin *atası* ilgisiz bir sebeple değiştirilirse
  imleç/seçim durumunu bozma riski taşıyorsa) kendi olay
  handler'larınızda `ErrSkipRender`'ı açıkça kullanın.
- [ ] **Senkron olayını debounce edin.** Her iki editör de, senkronizasyon
  `<textarea>`'sinin `g-input` bağlamasında `g-debounce`'u (varsayılan
  350ms) ayarlar — her tek tuş vuruşunda bir WS olayı (ve dolayısıyla
  bir `HandleEvent` çağrısı) göndermek, görünür bir render tetiklemese
  bile israftır; benzer herhangi bir serbest-yazma senkronizasyon
  kanalı için bir debounce'u yerinde tutun.

---

## 6. "Bileşen kodunu değiştirdim ama hiçbir şey değişmedi" (sunucu yeniden başlatma)

Belirti: bir bileşenin `Render`/`HandleEvent`/`Mount`'unu (veya başka
herhangi bir Go kaynağını) düzenlediniz, tarayıcı sekmesini yeniden
yüklediniz, ve uygulama hâlâ eski kod gibi davranıyor.

Bunun, GoUI'ye karşı geliştirme yaparken en yaygın yanlış alarm olduğu
için içselleştirilmeye değer tek bir nedeni vardır: **GoUI bileşenleri
derlenmiş Go kodudur, şablonlar veya yorumlanmış script'ler değil.**
`client/*.js` içindeki istemci tarafı JS'nin aksine (tarayıcının normal
bir sayfa yeniden yüklemesinde yeniden aldığı), herhangi bir `.go`
dosyasındaki bir değişiklik — bir bileşen, bir doğrulama kuralı, kayıtlı
bir factory, registry bağlantısının kendisi — etkili olmadan önce Go
process'inin **yeniden inşa edilmesi ve yeniden başlatılması** gerekir.
GoUI'nin kendi içinde hot-reload yoktur.

Kontrol listesi:

- [ ] **Gerçekten yeniden inşa mı ettiniz?** `go run ./cmd/myapp`, her
  çağrıda yeniden inşa eder; uzun süre çalışan bir
  `go build && ./bin/myapp` iş akışı bunu yapmaz — yineleme
  yapıyorsanız, her seferinde ya `go build`'i yeniden çalıştırın ya da
  kaydettiğinizde binary'i yeniden inşa eden *ve* yeniden başlatan bir
  dosya izleyici (`air`, `wgo`, `entr`, `reflex` vb.) kullanın.
- [ ] **Yeniden başlatılan process, vurduğunuz portu gerçekten bağladı
  mı** (önceki bir örnekten `"address already in use"` yok)? Eski bir
  process'i sessizce trafiği sunmaya devam ettiren başarısız bir
  yeniden başlatma, "değişikliğimin hiçbir etkisi olmadı"yla aynı
  görünür.
- [ ] **Mevcut tarayıcı sekmeleri, *eski* process tarafından barındırılan
  bir oturuma yeniden bağlanıp, sonra *yeni* olandan bir
  `"session not found"` mı aldı?** Yeniden başlatmadan sonra, bellek
  içi `*ws.Hub` boştur — önceden bağlı her tarayıcı sekmesinin
  `goui.sessionId`'si şimdi artık var olmayan bir oturumu ifade eder.
  §2'ye göre, istemci bunu otomatik olarak zaten kurtarır (bayat ID'yi
  temizler, taze yeniden bağlanır, yeniden mount eder). Taze bir mount
  gerçekleştiğini görmüyorsanız, sunucu tarafı değişikliğin etkili
  olmadığı sonucuna varmak yerine sekmeyi sert-yeniden yükleyin
  (herhangi bir önbelleğe alınmış JS'yi atlayarak).
- [ ] **`go.mod`'unuzun gerçekten import ettiği `goui` modülünün vendor
  edilmiş/kopyalanmış bir checkout'unu değil, gerçekten import edileni
  mi düzenliyorsunuz?** Uygulamanız `github.com/zatrano/goui`'ye bir
  versiyon pin'i aracılığıyla bağımlıysa (bir local `replace` yönergesi
  çalışma kopyanıza işaret etmiyorsa), ayrı bir yerel klondaki dosyaları
  düzenlemek, `go.mod`'unuza yerel geliştirme için
  `replace github.com/zatrano/goui => ../path/to/goui` eklemeden ya da
  bağımlılığı yükseltmeden/yeniden vendor etmeden binary üzerinde bir
  etkisi olmaz.
- [ ] **Sadece stilendirmeyi mi değiştirdiniz (`forms/style.css`, bir
  tema geçersiz kılması) ve bir Go yeniden inşasını mı bekliyorsunuz?**
  CSS ve `client/` altındaki istemci JS modülleri, olduğu gibi sunulan
  statik dosyalardır — bunlar düz bir tarayıcı yeniden yüklemesinde
  (veya tarayıcı onları agresif bir şekilde önbelleğe aldıysa sert bir
  yeniden yüklemede) *gerçekten* etkili olur. Bir CSS değişikliğini
  kovalarken Go process'ini yeniden başlatmayın; tersine, bir tarayıcı
  yeniden yüklemesinin tek başına bir Go tarafı değişikliği alacağını
  beklemeyin.
</contents>
