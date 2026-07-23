# 10. Diffing İçyapısı (Internals)

GoUI'deki her yeniden render aynı hattan geçer: yeni HTML'i render et,
onu küçük bir ağaca ayrıştır, önceki render'dan gelen ağaca karşı
diff'le, ve ortaya çıkan yama listesini WebSocket üzerinden gönder. Bu
belge, bu ağacı, yama formatını, ve diff algoritmasını, belirli bir
değişikliğin ürettiği yamaları neden ürettiğini anlayabileceğiniz ve
nerede dikkatli olmanız gerektiğini bilebileceğiniz kadar açıklar.

Bu belge boyunca kullanılan modül yolu: `github.com/zatrano/goui`. İlgili
paket `diff`'tir (`diff/node.go`, `diff/diff.go`, `diff/patch.go`,
`diff/serialize.go`).

## 1. `Node`

```go
// diff/node.go
type Node struct {
	Tag      string
	Text     string
	Attrs    map[string]string
	Key      string
	Children []*Node
}
```

- **Eleman düğümleri**, boş olmayan bir `Tag`'e (örn. `"div"`, `"li"`) ve
  `Attrs`'e sahiptir.
- **Metin düğümleri**, `Tag == ""`'a sahiptir (`isTextNode` tam olarak
  bunu kontrol eder) ve içeriklerini `Text`'te taşırlar. Sadece boşluk
  içeren metin düğümleri, ayrıştırma sırasında tamamen düşürülür
  (`convertHTMLNode` bunları atlar), böylece girintili, güzel
  biçimlendirilmiş Go string literal'ları hayalet metin düğümleri
  oluşturmaz.
- **`Key`**, ayrıştırma sırasında mevcutsa bir `data-key`
  özniteliğinden otomatik olarak doldurulur:

  ```go
  for _, attr := range n.Attr {
      node.Attrs[attr.Key] = attr.Val
      if attr.Key == "data-key" {
          node.Key = attr.Val
      }
  }
  ```

  `data-key`, `Attrs`'te de kalır (dolayısıyla DOM'da hâlâ normal bir
  HTML özniteliğidir) — `Key`, ondan türetilen sadece diff-zamanı bir
  kolaylıktır.

`diff.ParseHTML(htmlStr string) (*Node, error)`, bir parçayı
(`golang.org/x/net/html`'nin bir `<div>` bağlamıyla parça ayrıştırıcısını
kullanarak) `Children`'ı gerçek üst seviye düğümler olan sentetik bir
`Tag: "root"` düğümüne ayrıştırır. `diff.Serialize(node *Node) string`
tersini yapar — ağacı gezer ve HTML'i geri yazar, deterministik çıktı
için öznitelikleri alfabetik olarak sıralar (hem yamalardaki tam düğüm
HTML'i için hem de testler için kullanılır).

## 2. `Patch` ve `PatchOp`

```go
// diff/patch.go
type PatchOp string

const (
	OpReplace    PatchOp = "replace"
	OpUpdateText PatchOp = "update_text"
	OpSetAttr    PatchOp = "set_attr"
	OpRemoveAttr PatchOp = "remove_attr"
	OpInsert     PatchOp = "insert"
	OpRemove     PatchOp = "remove"
	OpMove       PatchOp = "move"
)

type Patch struct {
	Op      PatchOp `json:"op"`
	Path    []int   `json:"path"`
	Tag     string  `json:"tag,omitempty"`
	Text    string  `json:"text,omitempty"`
	Attr    string  `json:"attr,omitempty"`
	Value   string  `json:"value,omitempty"`
	HTML    string  `json:"html,omitempty"`
	Key     string  `json:"key,omitempty"`
	FromIdx int     `json:"from_idx,omitempty"`
	ToIdx   int     `json:"to_idx,omitempty"`
}
```

`Patch`, tel üzerinden gerçekten geçen şeydir (JSON olarak, bir
`"render"` frame'inin `payload`'ı içinde). Tek bir yeniden render sıfır,
bir, veya birçok yama üretebilir.

### 2.1 `Path`

`Path`, bir **çocuk indeksleri** listesidir, bir CSS seçicisi veya bir
DOM API yolu değil. *Bileşenin kök elemanından* başlayarak (bkz. §3
aşağıda), tekrar tekrar N'inci "anlamlı çocuğu" (bir eleman, veya
boş-olmayan bir metin düğümü; istemcinin `meaningfulChildren()` yardımcısı
sunucu tarafında ayrıştırıcının kullandığı tam olarak aynı filtreleme
kuralını uygular) alarak yürünür. `Path: []`, her zaman "kökün kendisi"
anlamına gelir.

### 2.2 Yedi op

| Op | Anlam | İlgili alanlar |
|---|---|---|
| `replace` | `Path`'teki düğümü taze serileştirilmiş HTML ile değiştir. Bir bileşenin çok ilk render'ı (`Path: []`) için ve bir düğümün etiketi değiştiğinde kullanılır. | `HTML`, `Tag`, `Key` |
| `update_text` | Bir metin düğümünün içeriğini yerinde değiştir (çevreleyen yapının yeniden ayrıştırılması olmadan). | `Text` |
| `set_attr` | `Path`'teki elemanda bir özniteliği ayarla/üzerine yaz. | `Attr`, `Value` |
| `remove_attr` | `Path`'teki elemandan bir özniteliği kaldır. | `Attr` |
| `insert` | `Path`'te yeni bir çocuk ekle (HTML olarak verilmiş), `Path`'teki son indeks ebeveynin çocukları arasındaki konum olsun. | `HTML`, `Tag`, `Key` |
| `remove` | `Path`'teki çocuğu kaldır. | `Key`, `Tag` |
| `move` | Mevcut, key'li bir çocuğu, ebeveyni içinde (ebeveyn `Path` ile tanımlanır) `FromIdx`'ten `ToIdx`'e, HTML'ine dokunmadan yeniden konumlandır. | `Key`, `FromIdx`, `ToIdx` |

İstemcinin yama uygulayıcısı (`client/goui.js`'deki `applyPatch`), bunların
her birini tam olarak açıklandığı gibi uygular, `checked`/`selected`/
`disabled`/`readOnly` üzerinde `set_attr`/`remove_attr` için boolean-özellik
senkronizasyonu dahil (bunun neden önemli olduğu için bkz.
[14-troubleshooting.md](14-troubleshooting.md)), ve bir şey uygulamadan
önce `data-goui-ignore`'u da kontrol eder — bkz. §6 ve
[11-file-uploads.md](11-file-uploads.md)/rich-text notları.

## 3. Yollar sayfaya değil bileşen köküne göre relatiftir

Bileşen nerede mount edildiğinden bağımsız olarak, `Path`'in her zaman
tek bir bileşene göre relatif olmasını sağlamak için iki şey birlikte
çalışır:

1. **`decorateComponentHTML`** (`ws/session.go`), `Render()`'ın
   döndürdüğü her ne ise `data-goui-component="<id>"` ile etiketler:
   - `Render()` tam olarak **bir** kök eleman döndürdüyse, o elemanın
     kendisi `data-goui-component` özniteliğini alır — sarmalayıcı
     eklenmez.
   - **Sıfır** çocuk döndürdüyse (boş string), bir yer tutucu
     `<div data-goui-component="<id>"></div>` kullanılır.
   - **Birden fazla kardeş** kök eleman döndürdüyse, bunlar sentetik bir
     `<div data-goui-component="<id>">...</div>` içine sarılır.

2. **`parseComponentTree`**, ardından `ParseHTML`'in her zaman ürettiği
   sentetik `Tag: "root"` düğümünü açar, ve — tam olarak bir üst seviye
   çocuk olduğunda — diff kökü olarak `"root"` sarmalayıcısı yerine *o
   çocuğu* (gerçek `data-goui-component` elemanı) kullanır:

   ```go
   // ws/session.go
   func parseComponentTree(html string) (*diff.Node, error) {
       tree, err := diff.ParseHTML(html)
       if err != nil {
           return nil, err
       }
       if len(tree.Children) == 1 {
           return tree.Children[0], nil
       }
       return tree, nil
   }
   ```

Sonuç: **`Render()`, tam olarak bir kök eleman döndürmelidir.** Öyle
yaparsa, o eleman hem diff için kullanılan ağaç kökü *hem de* istemcinin
DOM'da aradığı `[data-goui-component]` elemanı olur
(`this.mount.querySelector('[data-goui-component="..."]')`), dolayısıyla
`Path: [0]`, "sayfadaki ikinci üst seviye şey" veya "GoUI'nin markup'ınızın
etrafına eklediği sarmalayıcı div" değil, hep "benim ilk çocuğum" demektir.
`Render()` birden fazla kardeş döndürürse, GoUI hâlâ çalışır (sentetik
sarmalayıcı aracılığıyla), ama DOM'da ve her yol hesaplamasında ekstra,
aksi halde anlamsız bir `<div>` kazanırsınız — zararsız, ama fallback'e
güvenmek yerine çoklu-eleman çıktıyı kendi tek kök etiketinizle sararak
kaçınmaya değer.

## 4. Diff algoritması

`diff.Diff(old, new *Node) []Patch`, her iki ağacı da kilit adımda
(lock-step) gezer, `path = []`'ten başlayarak (`diff/diff.go`'daki
`diffNode`):

1. **Nil durumları** — `old == nil && new != nil` → `insert`;
   `old != nil && new == nil` → `remove`; ikisi de nil ise → hiçbir şey.
2. **Etiket uyuşmazlığı** — aynı konumda farklı `Tag` → tüm düğümü
   `replace` et (bu, eleman-metin-düğümü takaslarını da kapsar, çünkü
   bir metin düğümünün `Tag`'i her zaman `""`'dır).
3. **İkisi de metin düğümü** — `Text`'i karşılaştır; sadece gerçekten
   değiştiyse `update_text` yay.
4. **Aynı etiketli elemanlar** — önce öznitelikleri diff'le, sonra
   çocukları diff'le.

### 4.1 Öznitelik diff'leme

`diffAttrs`, eski ve yeni öznitelik key'lerinin **birleşimini**
(union), deterministik yama sıralaması için sıralanmış olarak hesaplar,
ve her key için:

- sadece `new`'de mevcut → `set_attr`
- sadece `old`'da mevcut → `remove_attr`
- her ikisinde de mevcut ama farklı değer → `set_attr`
- her ikisinde de mevcut, aynı değer → yama yok

### 4.2 Çocuk diff'leme: indeksli vs. key'li

`diffChildren`, *her ebeveyn düğüm için*, herhangi bir tarafın
çocuklarının bir `data-key` içerip içermediğine bağlı olarak iki
stratejiden birini seçer:

```go
func diffChildren(oldChildren, newChildren []*Node, path []int, patches *[]Patch) {
	if hasAnyKey(oldChildren) || hasAnyKey(newChildren) {
		diffKeyedChildren(oldChildren, newChildren, path, patches)
		return
	}
	diffIndexedChildren(oldChildren, newChildren, path, patches)
}
```

**İndeksli diff'leme** (bu çocuk listesinde hiçbir yerde key yok)
tamamen konumsaldır: ortak öneki (prefix) indeks-indeks diff'ler
(her çifte özyinelemeli olarak inerek), sonra herhangi bir ekstra yeni
kuyruk çocuğunu `insert` eder, sonra herhangi bir ekstra eski kuyruk
çocuğunu (sonundan başlayarak, böylece daha önceki indeksler geçerli
kalır) `remove` eder. Bu, sadece sonunda büyüyen/küçülen listeler için
etkili ve doğrudur, ama *ortadan* yeniden sıralanan bir liste için,
indeksli diff'leme "ilk farklı indeksten itibaren her şeyi
değiştir/güncelle"ye dejenere olur — key'li listelerin var olma
sebebinin tam olarak bu olması.

**Key'li diff'leme** — bkz. §5.

## 5. Key'li listeler: bir key→konum map'i, bir LCS değil

"Bu öğe konum 3'teydi ve şimdi konum 0'da" eşleştirme görevini
(rastgele yeniden sıralamalar için minimum yama sayısı ile) optimal
olarak çözmek, klasik olarak bir en-uzun-ortak-alt-dizi (LCS) algoritması
ile çözülür. GoUI LCS'i **uygulamaz**. `diffKeyedChildren` (`diff/diff.go`)
bunun yerine:

1. Önceden dört küçük map inşa eder: `oldByKey`/`oldPos` ve
   `newByKey`/`newPos`, gerçekten key'i olan çocuklar için `data-key`'den
   düğüme/indekse. Key'li bir listede key'siz çocuklar, key'li mantığın
   yanında satır içi olarak konumsal olarak diff'lenir.
2. `oldChildren`'ı **tersten** gezer ve key'i `newChildren`'da artık var
   olmayan herhangi bir key'li çocuk için bir `remove` yaması yayar.
3. Tek bir boolean belirler, `hasInsertOrRemove`: bir key bir tarafta
   var ama diğerinde yoksa true'dur (yani liste gerçekten üye kazandı
   veya kaybetti, sadece mevcut üyeleri yeniden sıralamadı).
4. `newChildren`'ı **ileri** gezer:
   - Key'siz yeni çocuk → eski listede aynı indekste her ne varsa ona
     karşı konumsal olarak diff'le (veya orada hiçbir şey yoksa
     `insert`).
   - Eşleşen eski key'i olmayan key'li yeni çocuk → `insert`.
   - Eşleşen eski key'i olan key'li yeni çocuk:
     - `!hasInsertOrRemove` **ve** konumu değiştiyse
       (`oldPos[key] != newPos[key]`), bir `move` yaması yay
       (`FromIdx: oldPos[key], ToIdx: newPos[key]`).
     - İçerik/öznitelik değişiklikleri hareket eden (veya sabit kalan)
       key'li bir öğede yakalanmaya devam etsin diye her zaman
       `diffNode(oldChild, newChild, ...)`'a da özyinelemeli olarak
       in.

Kısacası: **bu bir key→konum araması, bir alt dizi algoritması değil.**
Mevcut bir key için değişen her konum, kendi bağımsız `move` yaması
olur, *orijinal* önce/sonra konum map'lerinden hesaplanır (daha önceki
hareketler uygulanırken yeniden hesaplanmaz). Yaygın durumlar için —
ekleme, öne ekleme, bir öğeyi kaldırma, veya tek bir öğeyi taşıma —
tam olarak beklediğiniz yamaları üretir. **Eşzamanlı çok-öğeli yeniden
sıralamalar** için (örn. aynı anda birkaç çifti takas etme, veya tam bir
karıştırma), yayılan `move` yamaları, her biri uygulandıktan sonra çocuk
sırası değişen canlı bir DOM'a karşı sırayla uygulanır; indeksler herhangi
bir hareket uygulanmadan önce alınan anlık görüntülerden türetildiğinden,
birkaç eşzamanlı hareketin bir dizisi, uygun bir LCS-tabanlı reconciler'ın
yapacağı gibi minimal (veya hatta açıkça doğru görünen) bir DOM
operasyonları dizisine indirgenmesi garanti edilmez. Key'li listeleriniz
sürükle-bırak-yeniden-sıralama veya çoklu-seçim-ve-taşıma UI'sini
destekliyorsa bunu özellikle test edin, ve tek seferde bir konumu
değiştirmeyi (örn. "öğeyi bir yukarı/aşağı taşı" kontrolleri) veya ağır
yeniden sıralamada su sızdırmaz doğruluk gerekiyorsa tek bir render'dan
gelen birçok eşzamanlı `move` yamasına güvenmek yerine ebeveyn listenin
tam bir `replace`'ini yaymayı tercih edin.

## 6. `data-goui-ignore`

Key'lerden bağımsız olarak, `data-goui-ignore` ile işaretlenmiş herhangi
bir alt ağaç, yamalar uygulanırken **istemci** tarafından atlanır — bkz.
`client/goui.js`'deki `isGoUIIgnored`/`applyPatch`. Bu, DOM'u reconciler
tarafından asla dokunulmaması gereken istemciye ait widget'lar için
(Quill, CodeMirror) vardır; bunu sunucu tarafında `core.ErrSkipRender`
ile birleştirin, böylece sunucu bu widget'lara durumu yansıtan olaylar
için bir yama kümesi hesaplamayı bile denemez. Tam desen için
[14-troubleshooting.md](14-troubleshooting.md)'ye bakın.

## 7. Dinamik listeler için `data-key` — pratik rehberlik

Yeniden sıralanabilecek, filtrelenebilecek, veya ortadan eklenip
çıkarılabilecek (sadece bir uçtan büyüyen/küçülen listelere değil)
render ettiğiniz herhangi bir listenin tekrarlanan kök elemanına
`data-key="<stable-id>"` ekleyin:

```go
var b strings.Builder
b.WriteString("<ul>")
for _, item := range items {
	b.WriteString(`<li data-key="` + html.EscapeString(item.ID) + `">`)
	b.WriteString(html.EscapeString(item.Label))
	b.WriteString("</li>")
}
b.WriteString("</ul>")
```

Genel kurallar:

- **Key, aynı mantıksal öğe için render'lar arasında stabil olmalıdır**
  (bir veritabanı ID'si, bir dilim indeksi değil — bir dilim indeksi
  amacın tamamını yener, çünkü yeniden sıralamada değişen tam olarak
  odur).
- **Belirli bir çocuk listesinin tamamı veya hiçbiri** pratikte key
  taşımalıdır; `hasAnyKey`, bir çocuk bile bir `data-key`'e sahip olduğu
  anda *tüm* kardeş liste için key'li modu tetikler, dolayısıyla aksi
  halde key'siz bir listedeki bir `<li>` üzerindeki başıboş bir key,
  hepsi için (daha yavaş, hareket-farkında) key'li yolu zorlar.
- **Key'ler yalnızca o bir ebeveynin doğrudan çocuklarını etkiler.**
  İç içe listelerin kendi öğelerinde kendi `data-key`'lerine ihtiyacı
  vardır; key'ler yayılmaz.
- Sadece kuyrukta büyüyen key'siz listeler (sohbet mesajları, etkinlik
  akışları, "daha fazla yükle" sayfalama) indeksli diff'leme altında
  zaten optimaldir — orada key eklemeyin, yama-sayısı faydası olmadan
  öznitelik gürültüsü ekler.

## 8. Performans notları

- **Öznitelik diff'leme, düğüm başına O(key sayısı)'dır**, deterministik
  sıralama için bir sıralamayla — birçok özniteliği olan elemanlar için
  bile ihmal edilebilir.
- **İndeksli çocuk diff'leme, paylaşılan önek için O(min(eski, yeni))'dir**,
  artı kuyruk ekleme/kaldırma için O(|Δuzunluk|) — ucuzdur, ve bu,
  markup'ın büyük çoğunluğu için varsayılan yoldur (çoğu elemanın hiç
  key'li çocuğu yoktur).
- **Key'li çocuk diff'leme, map'leri inşa etmek için O(n) ve gezmek
  için O(n)'dir** — karesel (quadratic) bir patlama yok — ama *sayısı*
  en kötü durumda O(n)'e kadar olabilecek `move` yamaları üretir (her
  öğenin konumu değişti), ki bunların her biri istemci tarafından kendi
  DOM `insertBefore` çağrısı olarak uygulanır. Tam bir liste `replace`'i
  bir HTML string'i ve bir DOM takasıdır; ağır yeniden sıralanmış key'li
  bir liste onlarca küçük op'a dönüşebilir. Sık sık yeniden sıralanan
  çok büyük listeler (yüzlerce+ satır) için, her ikisini de ölçün ve
  listenin tamamının daha kaba bir `replace`'inin (key'leri bırakın,
  veya tam bir yeniden render'ı zorlayın) birçok `move` yamasından
  gerçekten daha ucuz olup olmadığını düşünün — GoUI bunu sizin için
  karar vermez.
- **`Serialize`, yalnızca `replace`/`insert` payload HTML'i için
  çağrılır** — sadece `set_attr`/`update_text`/`remove_attr`/`move`'a
  ihtiyaç duyan düğümler, alt ağaçlarını yeniden serileştirme maliyetini
  asla ödemez, ki bu, dar güncellemeleri (örn. sadece-metin bir sayaç),
  *etiketi* değişen bir elemana sık sık değişen içeriği sarmak yerine
  (bu, tam bir `replace`'i zorlar) tercih etmenin ana sebebidir.
- **Tüm hat, `core.ErrSkipRender` döndürmeyen her `HandleEvent`
  çağrısında bir kez çalışır.** Gürültülü istemci olaylarını
  (`g-debounce`, `forms/*` boyunca kullanılır) debounce edin, böylece
  hızlı bir daktilocu her tuş vuruşunda bir ayrıştır+diff'le+yamala
  döngüsünü tetiklemez — mevcut konvansiyon için (tipik olarak
  100–350ms) `TextInput`/`RichTextEditor` alan uygulamalarına bakın.
</contents>
