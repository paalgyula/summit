# Porting the Blizzard UI to the web client

How the in-game UI (character sheet, bags, action bars…) is defined by the
original WoW client, how we inspect it, and how we reproduce it as HTML/CSS in
`client/` using the asset server.

---

## 1. TL;DR

- The WoW 3.3.5a UI is **data-driven**: XML frames + Lua scripts, packaged in
  **MPQ** archives — not code we can read from the running client.
- There are two XML namespaces:
  - **GlueXML** — the *glue screens* (login, realm list, character select/create).
  - **FrameXML** — the *in-game UI* (character sheet, bags, action bars, HUD).
- Every frame has a **size** and an **anchor** (`point`/`relativeTo`/`relativePoint`
  + `offset`). We convert those anchors to absolute CSS `left`/`top` pixels.
- Every texture is a **BLP** referenced by an MPQ path (`Interface\...`). The
  **asset server** converts them to WebP and serves them at the same path with a
  `.webp` extension.
- The client references them with `assetUrl('Interface/...')`; nothing UI-related
  is bundled in `client/assets/` any more (only the procedural `sky/*` files).

---

## 2. GlueXML vs FrameXML (and what "Glue" is)

**"Glue"** is Blizzard's name for the *out-of-game* screens that hold the client
together before/around the world:

| Screen | Glue file |
|---|---|
| Login / account | `Interface\GlueXML\AccountLogin.xml` |
| Realm list | `Interface\GlueXML\RealmList.xml` |
| Character select | `Interface\GlueXML\CharacterSelect.xml` |
| Character create | `Interface\GlueXML\CharacterCreate.xml` |
| Race select | `Interface\GlueXML\RaceSelect.xml` |

Shared glue widgets live in `GlueButtons.xml`, `GlueTemplates.xml`,
`GlueBasicControls.xml`, `GlueFonts.xml`, `GlueDialog.xml`, …

The **glue textures** are under `Interface\Glues\...` (e.g.
`Interface\Glues\Common\Glues-WoW-WotLKLogo`, `Interface\Glues\CharacterCreate\*`,
`Interface\Glues\Models\UI_MainMenu_Northrend\*`).

**FrameXML** is the in-game UI. The frames we care about:

| Frame | File |
|---|---|
| Character menu / paper doll | `Interface\FrameXML\CharacterFrame.xml`, `PaperDollFrame.xml` |
| Bags / backpack | `Interface\FrameXML\ContainerFrame.xml` |
| Action bars | `Interface\FrameXML\ActionButton.xml`, `MainMenuBar*.xml` |
| Tooltips | `Interface\FrameXML\GameTooltip.xml` |

FrameXML textures are **not** under a `Glues` folder — they use their own roots
such as `Interface\PaperDoll\*`, `Interface\Buttons\*`, `Interface\ContainerFrame\*`,
`Interface\PaperDollInfoFrame\*`, `Interface\Icons\*`.

---

## 3. Where the UI actually lives: the MPQs

The XML and textures are inside the client's MPQ archives on the game server
(`vm3.pilab.hu:~/Client/Data`):

```
Data/common.MPQ, common-2.MPQ, expansion.MPQ, lichking.MPQ,
     patch.MPQ, patch-2.MPQ, patch-3.MPQ
Data/enUS/locale-enUS.MPQ, base-enUS.MPQ, patch-enUS-*.MPQ, …
```

Later patches override earlier archives, so when the same XML exists in several,
extract from the **highest** `patch-enUS-*` that contains it.

### Inspecting MPQs with `datagen`

`cmd/datagen` has an `mpq` subcommand (list / extract / batch BLP→WebP).

```bash
# Cross-compile for the Linux game host
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/datagen-linux ./cmd/datagen
scp bin/datagen-linux vm3.pilab.hu:~/datagen

# List everything indexed in an archive
ssh vm3.pilab.hu '~/datagen mpq --archive ~/Client/Data/enUS/locale-enUS.MPQ --list' \
  | grep -iE 'gluexml|paperdoll'

# Extract one file (BLP textures are transcoded to WebP with -f webp)
ssh vm3.pilab.hu '~/datagen mpq --archive ~/Client/Data/enUS/locale-enUS.MPQ \
  --extract "interface\framexml\paperdollframe.xml" -o /tmp/pd'

# Batch convert every BLP in an archive
ssh vm3.pilab.hu '~/datagen mpq --archive ~/Client/Data/common.MPQ --textures-to-webp -o /tmp/out'
```

MPQ paths use **backslashes** and are case-insensitive; the asset server is
case-insensitive too but **preserves the directory structure**.

---

## 4. The asset server: MPQ path → WebP

The asset server converts BLP → WebP and serves the result at the *same* MPQ
path with `.webp`:

| MPQ source | asset server URL |
|---|---|
| `Interface\PaperDoll\UI-PaperDoll-Slot-Head.blp` | `Interface/PaperDoll/UI-PaperDoll-Slot-Head.webp` |
| `Interface\PaperDollInfoFrame\UI-Character-StatBackground.blp` | `Interface/PaperDollInfoFrame/UI-Character-StatBackground.webp` |
| `Sound\Interface\iuimainmenubuttona.wav` | `Sound/Interface/iuimainmenubuttona.wav` |

Verify before wiring anything in:

```bash
curl -s -o /dev/null -w '%{http_code}\n' \
  https://assets-summit.dev.pilab.hu/Interface/PaperDoll/UI-PaperDoll-Slot-Head.webp
# 200
```

In the client, always build the URL with `assetUrl()` (`src/core/config.ts`) —
it prepends `ASSET_HOST` (default `https://assets-summit.dev.pilab.hu`,
override with `VITE_ASSET_HOST`) and appends the cache-busting `?v=ASSET_VERSION`.

```ts
import { assetUrl } from '../core/config';
const slotBg = assetUrl('Interface/PaperDoll/UI-PaperDoll-Slot-Head.webp');
```

---

## 5. The Blizzard layout model (frames, anchors, sizes)

Every `<Frame>`/`<Button>`/`<Texture>` has:

- **`<Size><AbsDimension x= y=/></Size>`** — width/height in UI pixels (the UI
  is authored for 1024×768; these are absolute pixels).
- **`<Anchors><Anchor point= relativeTo= relativePoint=><Offset><AbsDimension x= y=/></Offset></Anchor>`**
  — where the frame sits.

Anchor semantics:

- `point` is a point **on this element**; `relativePoint` is a point **on the
  reference** (the parent if `relativeTo` is omitted).
- `offset` is added in UI pixels. **+x is right, +y is UP** (screen coordinates
  in WoW are y-up, unlike CSS).
- A child anchored to the parent's `TOPLEFT` with offset `(x, -y)` is `x` px from
  the left and `y` px **down** from the top.
- A child anchored to the parent's `BOTTOMLEFT` with offset `(x, y)` is `x` px
  from the left and `y` px **up** from the bottom.

### Converting an anchor to CSS

CSS is top-left origin with y growing **down**. For a panel of height `H`:

| WoW anchor | CSS |
|---|---|
| `TOPLEFT` + `(x, -y)` | `left: x; top: y` |
| `BOTTOMLEFT` + `(x, y)` | `left: x; top: H - y` |
| `TOPLEFT relativeTo=<prev> TOPRIGHT` + `(dx, dy)` | `left: prev.left + prev.width + dx; top: prev.top - dy` |

**Worked example — the weapon row** (`PaperDollFrame.xml`):

```xml
<Button name="CharacterMainHandSlot" inherits="PaperDollItemSlotButtonTemplate">
  <Anchors>
    <Anchor point="TOPLEFT" relativePoint="BOTTOMLEFT">   <!-- parent = PaperDollFrame -->
      <Offset><AbsDimension x="122" y="127"/></Offset>
    </Anchor>
  </Anchors>
</Button>
```

The slot's **top-left** is 127 px **above** the frame's bottom-left, so:

```
top  = panelHeight - 127 = 512 - 127 = 385
left = 122
```

and the other two are chained:

```xml
<Anchor point="TOPLEFT" relativeTo="CharacterMainHandSlot" relativePoint="TOPRIGHT">
  <Offset><AbsDimension x="5" y="0"/></Offset>
```

```
OffHand.left = MainHand.left + slotWidth + 5 = 122 + 37 + 5 = 164
Ranged.left  = OffHand.left  + slotWidth + 5 = 206
```

> **Gotcha we hit:** `BOTTOMLEFT + (x, y)` places the element's **top** at
> `H - y`, *not* `H - y - height`. Subtracting the slot height too moves the row
> down by 37 px — this is exactly the "sword/shield/bow in the wrong position"
> bug.

Vertical stacks are usually a chain of `BOTTOMLEFT relativeTo=<prev> BOTTOMLEFT`
with `y = -4`, i.e. `next.top = prev.top + prev.height + 4` → a row pitch of
`37 + 4 = 41` px.

---

## 6. Textures: unique backgrounds, sprites, chrome

- **Each equipment slot has its own unique background texture**, not one shared
  slot image:

  ```
  Interface\PaperDoll\UI-PaperDoll-Slot-Head.blp
  Interface\PaperDoll\UI-PaperDoll-Slot-Neck.blp
  Interface\PaperDoll\UI-PaperDoll-Slot-Shoulder.blp
  Interface\PaperDoll\UI-PaperDoll-Slot-Rear.blp      (the Back slot)
  Interface\PaperDoll\UI-PaperDoll-Slot-Chest.blp
  … MainHand, SecondaryHand, Ranged, Finger, RFinger, Trinket, Waist, Legs,
    Feet, Hands, Wrists, Shirt, Tabard, Ammo, Relic, Bag
  ```

  An equipped item is drawn **on top** of that background (we use an `<img>` with
  a quality-coloured border).

- **Panel chrome is split into quadrants** and tiled. The character frame
  background is four textures:

  ```
  Interface\PaperDollInfoFrame\UI-Character-CharacterTab-L1         256×256  @ (0,0)
  Interface\PaperDollInfoFrame\UI-Character-CharacterTab-R1         128×256  @ (256,0)
  Interface\PaperDollInfoFrame\UI-Character-CharacterTab-BottomLeft 256×256  @ (0,256)
  Interface\PaperDollInfoFrame\UI-Character-CharacterTab-BottomRight128×256  @ (256,256)
  ```

  (L = left half, R = right half; 1 = top, Bottom* = lower half.)

- **Sprites / strips** are single images you index with `background-position`.
  `UI-Character-ResistanceIcons` is a 32×160 strip of five 32 px rows (schools
  of magic):

  ```ts
  backgroundImage: url(assetUrl('Interface/PaperDollInfoFrame/UI-Character-ResistanceIcons.webp')),
  backgroundPosition: `0 -${i * 29}px`,   // 145px strip → 5 rows of 29px
  ```

- **`alphaMode="ADD"`** on a texture means additive blending — reproduce with
  `mix-blend-mode: screen` / `plus-lighter` (we use it for the slot highlight).

---

## 7. Porting a frame, end to end (CharacterFrame / PaperDollFrame)

1. **Find the XML.** List and grep the archives for the frame:
   `characterframe.xml`, `paperdollframe.xml`, `characterframetemplates.xml`.
2. **Extract it** (`datagen mpq --extract …`) and read the frames:
   - `CharacterFrame` = 384×512 panel, tabs, name, close button, portrait.
   - `PaperDollFrame` = the content (model, stats, resistances, slots).
3. **Collect the textures** referenced by `file="Interface\…"`; map each to its
   `assetUrl(...)` and `curl`-check the `.webp` is 200.
4. **Convert the layout** to absolute CSS. Keep the numbers verbatim from the XML
   (`left`/`top` in px) so it stays faithful; put geometry in a CSS file and
   texture URLs in the component (CSS can't call `assetUrl`).
5. **Implement the React component.** See
   `client/src/ui/CharacterSheet.tsx` + `styles/CharacterSheet.css`:

   ```tsx
   const SLOT = 37;                 // ItemButtonTemplate
   const ROW = SLOT + 4;            // stacked slots are 4px apart
   const LEFT_SLOTS  = [{ slot: HEAD, x: 21, y: 74 }, { slot: NECK, x: 21, y: 74 + ROW }, …];
   const RIGHT_SLOTS = [{ slot: HANDS, x: 305, y: 74 }, …];
   const BOTTOM_Y = 512 - 127;      // MainHand/OffHand/Ranged
   ```

   Each slot renders its **own** background plus the equipped item icon:

   ```tsx
   const slotBg = assetUrl(`Interface/PaperDoll/UI-PaperDoll-Slot-${GHOST_ICON[slot]}.webp`);
   <div className="cs-slot" style={{ left: x, top: y, backgroundImage: `url('${slotBg}')` }} />
   ```

6. **Keep the data plumbing** (drag/drop, right-click unequip, tooltips) from the
   previous component — only the presentation changes.
7. **Build & eyeball**: `npm run build`, open the client, toggle the frame, and
   compare against a screenshot of the real client.

---

## 8. Don't bundle UI textures

Historically the client carried renamed copies in `client/assets/login/…`,
`charcreate/…`, etc. Those were just re-named extracts of `Interface\Glues\…`
(and `Interface\Buttons\…`, `Interface\PaperDoll\…`). Since the asset server
already serves the originals, the bundles were removed:

- TS/TSX: `localAssetUrl('login/x.webp')` → `assetUrl('Interface/Glues/…/X.webp')`.
- CSS: `url('/login/x.webp')` → `url('https://assets-summit.dev.pilab.hu/Interface/Glues/…/X.webp')`.
- `client/assets/` now holds only the procedural `sky/*.webp` (their MPQ origin
  is unclear), everything else is served remotely.

Rule of thumb: if a texture has an MPQ path, use `assetUrl`; only keep a file
local if it is genuinely generated by this project.

---

## 9. Checklist for porting the next frame

- [ ] Locate the FrameXML/GlueXML file and extract the latest patch version.
- [ ] Note every frame's `Size` and `Anchors`; convert with the table in §5.
- [ ] List every `file="Interface\…"` texture; `curl` each `.webp` for 200.
- [ ] Identify chrome quadrants vs. per-slot/per-element textures vs. sprites.
- [ ] Build the component: absolute-positioned divs, `assetUrl` textures,
      `mix-blend-mode` for `alphaMode="ADD"`, `background-position` for strips.
- [ ] Preserve interactions (tooltips, drag/drop, clicks) from the old component.
- [ ] `npx tsc --noEmit && npm run build`, then compare against the real client.
