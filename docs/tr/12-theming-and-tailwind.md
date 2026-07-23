# 12. Temalandırma ve Tailwind

GoUI'nin yerleşik form kontrollerinin her görsel yönü — renkler, köşe
yarıçapı, boşluk (spacing) — bir dosyada, bir kez tanımlanmış küçük bir
CSS özel özellikleri ("tasarım token'ları") kümesi tarafından
yönlendirilir. GoUI'yi yeniden markalamak sadece bir CSS egzersizidir;
görünümü değiştirmek için hiçbir zaman Go kodunu veya bileşen
şablonlarını düzenlemezsiniz.

Bu belge boyunca kullanılan modül yolu: `github.com/zatrano/goui`.

## 1. Token kümesi

Tüm token'lar `forms/style.css`'in en üstündeki `:root`'ta yaşar:

```css
:root {
  --color-goui-primary: oklch(55% 0.18 250);
  --color-goui-border: oklch(85% 0.01 250);
  --color-goui-error: oklch(55% 0.22 25);
  --color-goui-success: oklch(52% 0.14 145);
  --color-goui-warning: oklch(70% 0.15 75);
  --color-goui-info: oklch(55% 0.12 250);
  --color-goui-text: oklch(20% 0.01 250);
  --color-goui-surface: oklch(99% 0.005 250);
  --radius-goui: 0.375rem;
  --spacing-goui-field: 0.75rem;
}
```

| Token | Ne için kullanılır | Tüketiciler |
|---|---|---|
| `--color-goui-primary` | Düğmeler, odak halkaları, vurgu girdileri, seçili/aktif durumlar | `.bg-goui-primary`, `.accent-goui-primary`, takvim gün seçimi, çipler, swatch'lar |
| `--color-goui-border` | Girdiler, fieldset'ler, paneller üzerindeki varsayılan kenarlıklar | `.border-goui-border`, çoğu `goui-*` bileşen kenarlığı |
| `--color-goui-error` | Geçersiz alan durumu, hata toast'ları | `.text-goui-error`, `.border-goui-error`, `.goui-toast-error`, parola gücü "weak" |
| `--color-goui-success` | Başarı toast'ları | `.goui-toast-success` |
| `--color-goui-warning` | Uyarı toast'ları | `.goui-toast-warning` |
| `--color-goui-info` | Bilgi toast'ları (aynı zamanda fallback/varsayılan tür) | `.goui-toast-info` |
| `--color-goui-text` | Gövde/etiket metin rengi, ve çoğu `color-mix()`'ten türetilmiş tonun (soluk metin, hover tonları, gölgeler) temeli | `.text-goui-text`, etiketler, yardımcı metin, gölgeler |
| `--color-goui-surface` | Girdilerin, panellerin, dropdown'ların, toast'ların arka planları | `.goui-input`, `.goui-searchable-panel`, `.goui-toast` |
| `--radius-goui` | Neredeyse her kontrol boyunca köşe yarıçapı | `.rounded-goui`, düğmeler, çipler, takvim hücreleri |
| `--spacing-goui-field` | Yatay/dikey alan dolgusu ve boşlukları | `.px-goui-field`, `.py-goui-field`, `.gap-goui-field` |

Paletin şekli hakkında dikkat edilmesi gereken iki şey:

- Renkler **`oklch()`**'te tanımlanır (lightness%, chroma, hue), bu
  yüzden stil sayfasının kalanının çoğu, her tonaj için ayrı değişkenler
  hard-code etmek yerine hover/muted/tint tonlarını türetmek için
  `color-mix(in oklch, var(--color-goui-...) X%, white)`'i katmanlar —
  algısal olarak tek biçimli açma/koyulaştırma `oklch` ile neredeyse
  bedava gelir.
- Yukarıdaki token'lar yalnızca GoUI'nin kendi bileşenleri için yük
  taşıyıcıdır (load-bearing). `.w-full`, `.flex`, `.text-sm` gibi
  yardımcı sınıflar (bunlar da `forms/style.css`'te) düz, sabit CSS'tir,
  token'laştırılmamıştır — paketlenmiş örneklerin kutudan çıktığı gibi
  makul görünmesi için varlar, tam bir Tailwind build'i gerekmeden
  (üzerine gerçek Tailwind'i bağlamak için §3'e bakın).

## 2. Kendi markanız için token'ları geçersiz kılma

Her bileşen token'ları `var(--color-goui-...)` aracılığıyla okuduğundan,
yeniden markalama sadece aynı özel özellik isimlerini, CSS cascade'inde
`forms/style.css` yüklendikten **sonraki** bir noktada yeniden bildirmek
demektir (daha sonraki aynı-özgüllük (specificity) bir `:root` kuralı
kazanır, veya daha yüksek özgüllük gerekiyorsa bunu `body`'ye/bir
sarmalayıcı sınıfına kapsamlayın).

Çalışan örnek — yeşil bir primary ve daha sıcak bir nötr paletle hayali
bir "RenewOS" markası:

```css
/* renewos-theme.css — bunu forms/style.css'ten SONRA yükleyin */
:root {
  --color-goui-primary: oklch(58% 0.16 155);   /* RenewOS yeşili */
  --color-goui-border:  oklch(88% 0.01 90);    /* sıcak açık gri */
  --color-goui-error:   oklch(56% 0.21 22);
  --color-goui-success: oklch(60% 0.15 150);
  --color-goui-warning: oklch(74% 0.14 70);
  --color-goui-info:    oklch(60% 0.10 220);
  --color-goui-text:    oklch(22% 0.015 90);
  --color-goui-surface: oklch(98% 0.006 90);
  --radius-goui: 0.5rem;               /* varsayılan 0.375rem'den biraz daha yuvarlak */
  --spacing-goui-field: 0.875rem;      /* biraz daha nefes alma alanı */
}
```

```html
<link rel="stylesheet" href="/forms/style.css">
<link rel="stylesheet" href="/assets/renewos-theme.css">
```

Bu, yeniden markalamanın tamamı budur: `forms/*` altındaki her düğme,
girdi kenarlığı, toast, takvim seçimi, çip ve odak
halkası, hiçbir bileşenin Go koduna veya render edilmiş markup'ına
dokunmadan hemen RenewOS paletini alır. Bunu daha da kapsamlayabilirsiniz
— örn. bir sarmalayıcı elemanda `.theme-renewos { --color-goui-primary: ...; }` —
bir tek sayfanın aynı anda birden fazla temayı barındırması gerekiyorsa
(örneğin beyaz-etiketli çok kiracılı bir admin paneli).

Sadece gerçekten değiştirmek istediğiniz token'ları geçersiz kılın;
yeniden bildirmediğiniz her şey `forms/style.css`'ten varsayılan değerini
korur.

## 3. Tailwind entegrasyonu

`forms/style.css` ve token'ları **hiç Tailwind olmadan** da mükemmel
çalışır — yukarıdaki düz `<link rel="stylesheet">` yaklaşımının yaptığı
şey budur. Tailwind tamamen isteğe bağlıdır ve sadece *kendi*
uygulama markup'ınızı (GoUI form kontrollerinin dışında) aynı marka
token'larını paylaşan Tailwind yardımcı sınıflarıyla yazmak istiyorsanız
faydalıdır.

GoUI'nin çekirdeğinin **npm bağımlılığı ve `package.json`'ı yoktur** —
Tailwind, kullanılırsa, sadece bir örnek/uygulama dizininin içinde,
Tailwind v4'ün standalone CLI'sini `npx` aracılığıyla (yerel bir npm
projesi gerekmeden) kullanılarak getirilir:

```css
/* examples/contact-form/input.css */
@import "../../forms/style.css";

@theme {
  --color-goui-primary: oklch(55% 0.18 250);
  --color-goui-border: oklch(85% 0.01 250);
  --color-goui-error: oklch(55% 0.22 25);
  --color-goui-text: oklch(20% 0.01 250);
  --radius-goui: 0.375rem;
  --spacing-goui-field: 0.75rem;
}
```

```bat
:: examples/contact-form/build-css.bat
@echo off
REM İsteğe bağlı: input.css'ten Tailwind yardımcılarını inşa et
REM Gerekli: npx (veya standalone tailwindcss binary)
npx --yes @tailwindcss/cli@4 -i ./input.css -o ./output.css
echo Built output.css — point index.html at it if you want Tailwind-generated utilities.
```

Çalıştırın (Windows):

```powershell
cd examples\contact-form
.\build-css.bat
```

ki bu, doğrudan çalıştırılabilecek, platformlar arası olarak eşdeğerdir:

```bash
npx --yes @tailwindcss/cli@4 -i ./input.css -o ./output.css
```

Burada neler oluyor:

1. `@import "../../forms/style.css";`, önce GoUI'nin temel kurallarını
   ve varsayılan token'larını çeker.
2. `@theme { ... }` bloğu, Tailwind v4'ün tasarım token'larını
   **birinci sınıf Tailwind tema değerleri** olarak kaydetme
   mekanizmasıdır — `--color-goui-primary`'yi `@theme` içinde
   bildirmek, Tailwind'in onun için de eşleşen yardımcı sınıflar
   üretmesi anlamına gelir (örn. `bg-[color:var(--color-goui-primary)]`-
   tarzı erişim, veya token'ı Tailwind'in kendi adlandırma
   konvansiyonu altında yansıtırsanız, `bg-goui-primary` gibi düz
   yardımcılar). Bu, hem GoUI'nin kendi kontrolleri hem de etraflarındaki
   herhangi bir elle yazılmış Tailwind markup'ı için marka renkleri
   üzerinde tek bir doğruluk kaynağını nasıl koruyacağınızdır.
3. `@tailwindcss/cli@4`, `npx --yes` aracılığıyla çağrılır, ki bu
   paketi talep üzerine indirir ve çalıştırır — **`npm install` yok,
   `node_modules` yok, repoda hiçbir yerde `package.json` yok.**
   Derleme zamanında ağa erişen `npx`'e bağımlı olmak istemiyorsanız,
   platformunuz için bir kez [standalone Tailwind CLI binary'sini](https://tailwindcss.com)
   indirin ve `npx --yes @tailwindcss/cli@4` yerine onu doğrudan çağırın;
   `-i`/`-o` argümanları her iki durumda da aynıdır.
4. Üretilen `output.css`, statik bir dosyadır — onu kendi
   build/deploy adımınızın bir parçası olarak commit edin veya yeniden
   üretin; GoUI'nin kendisi Tailwind'i veya Node'u çalışma zamanında,
   testlerde, veya `go build`'in herhangi bir yerinde asla çağırmaz.

### 3.1 Kendi markanızı Tailwind yolu üzerinden uygulama

Yeniden markalamak *ve* Tailwind kullanmayı sürdürmek için, kendi
`input.css`'inizin `@theme` bloğu içindeki değerleri değiştirin (§2'deki
düz-CSS geçersiz kılmayı yansıtarak), sonra yeniden inşa edin:

```css
/* input.css, RenewOS varyantı */
@import "../../forms/style.css";

@theme {
  --color-goui-primary: oklch(58% 0.16 155);
  --color-goui-border:  oklch(88% 0.01 90);
  --color-goui-error:   oklch(56% 0.21 22);
  --color-goui-text:    oklch(22% 0.015 90);
  --radius-goui: 0.5rem;
  --spacing-goui-field: 0.875rem;
}
```

```bash
npx --yes @tailwindcss/cli@4 -i ./input.css -o ./output.css
```

Düz `forms/style.css` yerine (veya ondan sonra) `output.css`'i sunun,
ve hem GoUI'nin yerleşik kontrolleri hem de kendi şablonlarınızdaki
herhangi bir Tailwind yardımcı sınıfı RenewOS paletini paylaşır.

## 4. Öneriler

- **Tailwind olmadan başlayın.** `forms/style.css`'i artı kendi token
  geçersiz kılmalarınızı bağlayın. Tailwind'i, GoUI'nin bileşenlerinin
  etrafında yardımcı sınıflar istediğiniz kadar özel uygulama markup'ı
  yazmaya başladığınızda ekleyin.
- **`forms/style.css`'in kendisini düzenlemek yerine marka/kiracı
  başına bir token geçersiz kılma dosyası tutun**, `forms/style.css`'ten
  sonra yüklenmiş — bu, temel stil sayfası değiştiğinde sizi önemsiz
  bir şekilde yükseltilebilir tutar.
- **Stilendirmeyi değiştirmek için bileşen Go kodunu fork'lamayın.**
  Koşullu stilendirmeye ihtiyaç duyan (geçersiz durum, devre dışı durum,
  seçim) her `forms/*` kontrolü, bunu zaten sınıflar/token'lar
  aracılığıyla ifade eder (`FieldValidation.ApplyErrorState`,
  `border-goui-border`/`border-goui-error` vb.) — sadece bir rengi
  değiştirmek için bir `.go` dosyasını düzenlemek istediğinizi
  fark ederseniz, önce token'a bakın.
</contents>
