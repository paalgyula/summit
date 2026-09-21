# Character Rendering — Customization, Equipment, Attachments

How the web client dresses a WoW 3.3.5a character, and what the servers
provide for it.

## Data flow

```
DBFilesClient\*.dbc ──(asset server, pkg/converter/dbc)──► GET /dbc/character.json
CharStartOutfit.dbc + Item.dbc ──(datagen dbc)──► summit.dat ──► world server
                                                     │
                                  SMSG_CHAR_ENUM: 23 slots × (displayId, inventoryType, enchant)
                                                     ▼
Character\<Race>\<Sex>\<Race><Sex>.m2 ──(asset server, pkg/converter/m2)──► .glb with Attach_<id> nodes
Item\ObjectComponents\<Weapon|Shield|Shoulder|Head>\*.m2 ──► .glb (texture type 2 replaced by the item texture)
Item\TextureComponents\<Region>Texture\<name>_<M|F|U>.blp ──► .webp
```

### Asset server: `dbc/character.json`

`pkg/converter/dbc` reads the raw WDBC tables and `BuildCharacterData`
assembles one JSON document (cached like every other conversion):

| Table | Used for |
|-------|----------|
| `ChrRaces` | race names and the helmet model suffix (`_HuM`, `_OrF`, ...) |
| `CharSections` | skin / face / facial hair / hair / underwear textures per race, sex, variation and colour; the flags decide what character creation offers |
| `CharHairGeosets` | hair style → geoset of group 0 (0 = bald) |
| `CharacterFacialHairStyles` | facial hair style → variants of geoset groups 1, 2, 3 |
| `HelmetGeosetVisData` | race masks of the groups a helmet hides (hair, facial 1‑3, ears) |
| `ItemDisplayInfo` | models, model textures, geoset groups, helmet visibility and the eight body texture components of every drawable item |
| `Item` | `SheatheType` per display id (where a sheathed weapon hangs) |

The asset server needs the DBCs: from the MPQs (`--mpq`), the asset dir
(`DBFilesClient/*.dbc`), or the upstream server (`--upstream`).

### World server

* `datagen dbc` now also writes the `Item.dbc` templates (entry → display id,
  inventory type, sheathe type) into `summit.dat`; the world server loads it
  with `world.WithStaticBaseData()` (a missing file only logs a warning, but
  then character creation fails with `CHAR_CREATE_FAILED` and no equipment is
  shown).
* `Player.InitInventory` places the `CharStartOutfit` items by inventory type
  (a second one‑hander goes to the off hand, the rest to the backpack).
* `SMSG_CHAR_ENUM` writes the template's display id and inventory type for
  the 19 equipment slots and the 4 bag slots (`INVENTORY_SLOT_BAG_END` = 23,
  verified against a captured 3.3.5a packet).

### M2 converter

* Attachments (`M2Attachment`, header offset 240) are exported as empty
  nodes `Attach_<id>` parented to their bone, with `extras.attachmentId`.
  Ids: 0 shield, 1 right hand, 2 left hand, 5/6 shoulders, 11 helm, 26/27
  back sheaths, 28 shield sheath, 32/33 hips.
* Texture transforms export rotation (`uvRotKeys`, unwrapped radians) and
  scale (`uvScaleKeys`) keys next to the translation keys.
* More sequences are exported by default: `Dead`, `Attack1H`, `Attack2H`,
  `ReadyUnarmed`, `Ready1H`, `Ready2H`, `Fall`, `SwimIdle`, `Swim`.
* BLP: textures with alpha depth 0 decode opaque (their palette alpha is 0,
  which made every composited skin transparent).

## Client (`client/src/entities`)

* `CharacterData.ts` fetches `dbc/character.json` once and answers the
  lookups; `customizationLimits()` gives the stepper ranges of the create
  screen from the `CharSections` flags (creation‑only, no barber shop styles,
  death knight skins only for death knights).
* `CharacterTexture.ts` composes the 512×512 body texture on a canvas: skin,
  face, facial hair, scalp, underwear, then the items' components in slot
  order (shirt, legs, feet, chest, wrists, hands, tabard, waist), each into
  its region of the 256‑based layout. Components try `_M`/`_F` before `_U`.
* `CharacterModel.ts` — `CharacterInstance`:
  * picks one variant per geoset group from hair / facial hair / items
    (sleeves 8, chest 10, trousers 13 for robes, pants 11, kneepads 9, boots
    5, gloves 4, belt 18, tabard 12, cape 15, ears 7, eyeglow 17 for DKs),
    helmets hide groups per race mask;
  * assigns the replaceable textures by M2 texture type (1 body, 6 hair,
    2 cape / item object skin, 8 skin extra);
  * loads helmet, shoulder, shield and weapon GLBs onto the attachment
    nodes; `setSheathed()` moves weapons between the hands and the sheath
    attachments (`Z` in the world), the idle clip becomes `Ready1H/2H`.
  * Re‑dressing keeps the loaded model when race and gender are unchanged.

## Dev notes

* Conversions are cached: after changing the converters, delete the stale
  files (`.dev/asset-cache/**/*.glb` for attachments, `**/*.webp` for the
  alpha fix). The deployed upstream server needs the same cache purge.
* Browsers cache assets for a year (`immutable`); use a hard reload or a
  fresh profile after purging the server cache.
* Without MPQs or an upstream the dev asset server only has the few raw
  files under `client/assets`: start it with `WOW_DATA=...` or
  `ASSET_UPSTREAM=https://assets-summit.dev.pilab.hu scripts/dev.sh`.
