package world

// SpellSchoolMask defines the spell school (elemental type).
type SpellSchoolMask uint32

// ShapeshiftForm defines the shapeshift form a unit can be in.
type ShapeshiftForm uint32

const (
	ShapeshiftFormNone            ShapeshiftForm = 0
	ShapeshiftFormCat             ShapeshiftForm = 1
	ShapeshiftFormTree            ShapeshiftForm = 2
	ShapeshiftFormTravel          ShapeshiftForm = 3
	ShapeshiftFormAquatic         ShapeshiftForm = 4
	ShapeshiftFormBear            ShapeshiftForm = 5
	ShapeshiftFormAmbient         ShapeshiftForm = 6
	ShapeshiftFormGhoul           ShapeshiftForm = 7
	ShapeshiftFormDireBear        ShapeshiftForm = 8
	ShapeshiftFormCreatureBear    ShapeshiftForm = 14
	ShapeshiftFormCreatureCat     ShapeshiftForm = 15
	ShapeshiftFormGhostWolf       ShapeshiftForm = 16
	ShapeshiftFormFlightForm      ShapeshiftForm = 27
	ShapeshiftFormSwiftFlightForm ShapeshiftForm = 28
	ShapeshiftFormStealth         ShapeshiftForm = 30
)

const (
	SpellSchoolMaskNone     SpellSchoolMask = 0
	SpellSchoolMaskPhysical SpellSchoolMask = 1
	SpellSchoolMaskHoly     SpellSchoolMask = 2
	SpellSchoolMaskFire     SpellSchoolMask = 4
	SpellSchoolMaskNature   SpellSchoolMask = 8
	SpellSchoolMaskFrost    SpellSchoolMask = 16
	SpellSchoolMaskShadow   SpellSchoolMask = 32
	SpellSchoolMaskArcane   SpellSchoolMask = 64

	SpellSchoolMaskAllMagic = SpellSchoolMaskFire | SpellSchoolMaskNature | SpellSchoolMaskFrost | SpellSchoolMaskShadow | SpellSchoolMaskArcane
	SpellSchoolMaskAll      = SpellSchoolMaskPhysical | SpellSchoolMaskAllMagic | SpellSchoolMaskHoly
)

// SpellEffect defines what a spell does.
// Values from AzerothCore SharedDefines.h SPELL_EFFECT_*.
type SpellEffect uint32

const (
	SpellEffectNone                          SpellEffect = 0
	SpellEffectInstantKill                   SpellEffect = 1
	SpellEffectSchoolDamage                  SpellEffect = 2
	SpellEffectDummy                         SpellEffect = 3
	SpellEffectPortalTeleport                SpellEffect = 4
	SpellEffectTeleportUnit                  SpellEffect = 5
	SpellEffectApplyAura                     SpellEffect = 6
	SpellEffectEnvironmentalDamage           SpellEffect = 7
	SpellEffectPowerDrain                    SpellEffect = 8
	SpellEffectHealthLeech                   SpellEffect = 9
	SpellEffectHeal                          SpellEffect = 10
	SpellEffectBind                          SpellEffect = 11
	SpellEffectPortal                        SpellEffect = 12
	SpellEffectRitualBase                    SpellEffect = 13
	SpellEffectRitualSpecialize              SpellEffect = 14
	SpellEffectRitualActivatePortal          SpellEffect = 15
	SpellEffectQuestComplete                 SpellEffect = 16
	SpellEffectWeaponDamageNOSchool          SpellEffect = 17
	SpellEffectResurrect                     SpellEffect = 18
	SpellEffectAddExtraAttacks               SpellEffect = 19
	SpellEffectDodge                         SpellEffect = 20
	SpellEffectEvade                         SpellEffect = 21
	SpellEffectParry                         SpellEffect = 22
	SpellEffectBlock                         SpellEffect = 23
	SpellEffectCreateItem                    SpellEffect = 24
	SpellEffectWeapon                        SpellEffect = 25
	SpellEffectDefense                       SpellEffect = 26
	SpellEffectPersistentAreaAura            SpellEffect = 27
	SpellEffectSummon                        SpellEffect = 28
	SpellEffectLeap                          SpellEffect = 29
	SpellEffectEnergize                      SpellEffect = 30
	SpellEffectWeaponPercentDamage           SpellEffect = 31
	SpellEffectTriggerMissile                SpellEffect = 32
	SpellEffectOpenLock                      SpellEffect = 33
	SpellEffectSummonChangeItem              SpellEffect = 34
	SpellEffectApplyAreaAuraParty            SpellEffect = 35
	SpellEffectLearnSpell                    SpellEffect = 36
	SpellEffectSpellDefense                  SpellEffect = 37
	SpellEffectDispel                        SpellEffect = 38
	SpellEffectLanguage                      SpellEffect = 39
	SpellEffectDualWield                     SpellEffect = 40
	SpellEffectJump                          SpellEffect = 41
	SpellEffectJumpDest                      SpellEffect = 42
	SpellEffectTeleportUnitsFaceCaster       SpellEffect = 43
	SpellEffectSkillStep                     SpellEffect = 44
	SpellEffectAddHonor                      SpellEffect = 45
	SpellEffectSpawn                         SpellEffect = 46
	SpellEffectTradeSkill                    SpellEffect = 47
	SpellEffectStealth                       SpellEffect = 48
	SpellEffectDetect                        SpellEffect = 49
	SpellEffectTransDoor                     SpellEffect = 50
	SpellEffectForceCriticalHit              SpellEffect = 51
	SpellEffectGuaranteeHit                  SpellEffect = 52
	SpellEffectEnchantItemTemporary          SpellEffect = 53
	SpellEffectTameCreature                  SpellEffect = 54
	SpellEffectSummonPet                     SpellEffect = 55
	SpellEffectLearnPetSpell                 SpellEffect = 56
	SpellEffectWeaponDamage                  SpellEffect = 57
	SpellEffectCreateRandomItem              SpellEffect = 58
	SpellEffectProficiency                   SpellEffect = 59
	SpellEffectSendEvent                     SpellEffect = 60
	SpellEffectPowerBurn                     SpellEffect = 61
	SpellEffectThreat                        SpellEffect = 62
	SpellEffectTriggerSpell                  SpellEffect = 63
	SpellEffectApplyAreaAuraRaid             SpellEffect = 64
	SpellEffectCreateManaGem                 SpellEffect = 65
	SpellEffectHealMaxHealth                 SpellEffect = 66
	SpellEffectInterruptCast                 SpellEffect = 67
	SpellEffectDistract                      SpellEffect = 68
	SpellEffectPull                          SpellEffect = 69
	SpellEffectPickpocket                    SpellEffect = 70
	SpellEffectAddFarsight                   SpellEffect = 71
	SpellEffectUntrainTalents                SpellEffect = 72
	SpellEffectApplyGlyph                    SpellEffect = 73
	SpellEffectHealMechanical                SpellEffect = 74
	SpellEffectSummonObjectWild              SpellEffect = 75
	SpellEffectScriptEffect                  SpellEffect = 76
	SpellEffectAttack                        SpellEffect = 77
	SpellEffectSanctuary                     SpellEffect = 78
	SpellEffectAddComboPoints                SpellEffect = 79
	SpellEffectCreateHouse                   SpellEffect = 80
	SpellEffectBindSight                     SpellEffect = 81
	SpellEffectDuel                          SpellEffect = 82
	SpellEffectStuck                         SpellEffect = 83
	SpellEffectSummonPlayer                  SpellEffect = 84
	SpellEffectActivateObject                SpellEffect = 85
	SpellEffectGameObjectDamage              SpellEffect = 86
	SpellEffectGameObjectRepair              SpellEffect = 87
	SpellEffectGameObjectSetDestructionState SpellEffect = 88
	SpellEffectKillCredit                    SpellEffect = 89
	SpellEffectThreatAll                     SpellEffect = 90
	SpellEffectEnchantHeldItem               SpellEffect = 91
	SpellEffectForceDeselect                 SpellEffect = 92
	SpellEffectSelfResurrect                 SpellEffect = 93
	SpellEffectSkinning                      SpellEffect = 94
	SpellEffectCharge                        SpellEffect = 95
	SpellEffectCastButton                    SpellEffect = 96
	SpellEffectKnockBack                     SpellEffect = 97
	SpellEffectDisenchant                    SpellEffect = 98
	SpellEffectInebriate                     SpellEffect = 99
	SpellEffectFeedPet                       SpellEffect = 100
	SpellEffectDismissPet                    SpellEffect = 101
	SpellEffectReputation                    SpellEffect = 102
	SpellEffectSummonObjectSlot1             SpellEffect = 103
	SpellEffectSummonObjectSlot2             SpellEffect = 104
	SpellEffectSummonObjectSlot3             SpellEffect = 105
	SpellEffectSummonObjectSlot4             SpellEffect = 106
	SpellEffectDispelMechanic                SpellEffect = 107
	SpellEffectResurrectPet                  SpellEffect = 108
	SpellEffectDestroyAllTotems              SpellEffect = 109
	SpellEffectDurabilityDamage              SpellEffect = 110
	SpellEffect_111                          SpellEffect = 111
	SpellEffectResurrectNew                  SpellEffect = 112
	SpellEffectAttackMe                      SpellEffect = 113
	SpellEffectDurabilityDamagePct           SpellEffect = 114
	SpellEffectSkinPlayerCorpse              SpellEffect = 115
	SpellEffectSpiritHeal                    SpellEffect = 116
	SpellEffectSkill                         SpellEffect = 117
	SpellEffectApplyAreaAuraPet              SpellEffect = 118
	SpellEffectTeleportGraveyard             SpellEffect = 119
	SpellEffectNormalizedWeaponDmg           SpellEffect = 120
	SpellEffect_121                          SpellEffect = 121
	SpellEffectSendTaxi                      SpellEffect = 122
	SpellEffectPullTowards                   SpellEffect = 123
	SpellEffectModifyThreatPercent           SpellEffect = 124
	SpellEffectStealBeneficialBuff           SpellEffect = 125
	SpellEffectProspecting                   SpellEffect = 126
	SpellEffectApplyAreaAuraFriend           SpellEffect = 127
	SpellEffectApplyAreaAuraEnemy            SpellEffect = 128
	SpellEffectRedirectThreat                SpellEffect = 129
	SpellEffectPlaySound                     SpellEffect = 130
	SpellEffectPlayMusic                     SpellEffect = 131
	SpellEffectUnlearnSpecialization         SpellEffect = 132
	SpellEffectKillCredit2                   SpellEffect = 133
	SpellEffectCallPet                       SpellEffect = 134
	SpellEffectHealPct                       SpellEffect = 135
	SpellEffectEnergizePct                   SpellEffect = 136
	SpellEffectLeapBack                      SpellEffect = 137
	SpellEffectClearQuest                    SpellEffect = 138
	SpellEffectForceCast                     SpellEffect = 139
	SpellEffectForceCastWithValue            SpellEffect = 140
	SpellEffectTriggerSpellWithValue         SpellEffect = 141
	SpellEffectApplyAreaAuraOwner            SpellEffect = 142
	SpellEffectKnockBackDest                 SpellEffect = 143
	SpellEffectPullTowardsDest               SpellEffect = 144
	SpellEffectActivateRune                  SpellEffect = 145
	SpellEffectQuestFail                     SpellEffect = 146
	SpellEffectTriggerMissileSpellWithValue  SpellEffect = 147
	SpellEffectChargeDest                    SpellEffect = 148
	SpellEffectQuestStart                    SpellEffect = 149
	SpellEffectTriggerSpell2                 SpellEffect = 150
	SpellEffectSummonRafFriend               SpellEffect = 151
	SpellEffectCreateTamedPet                SpellEffect = 152
	SpellEffectDiscoverTaxi                  SpellEffect = 153
	SpellEffectTitanGrip                     SpellEffect = 154
	SpellEffectEnchantItemPrismatic          SpellEffect = 155
	SpellEffectCreateItem2                   SpellEffect = 156
	SpellEffectMilling                       SpellEffect = 157
	SpellEffectAllowRenamePet                SpellEffect = 158
	SpellEffectForceCast2                    SpellEffect = 159
	SpellEffectTalentSpecCount               SpellEffect = 160
	SpellEffectTalentSpecSelect              SpellEffect = 161
	SpellEffect_162                          SpellEffect = 162
	SpellEffectRemoveAura                    SpellEffect = 163
)

// AuraType defines what kind of buff/debuff an aura applies.
// Values from AzerothCore SpellAuraDefines.h SPELL_AURA_*.
type AuraType uint32

const (
	AuraNone                                AuraType = 0
	AuraBindSight                           AuraType = 1
	AuraModPossess                          AuraType = 2
	AuraPeriodicDamage                      AuraType = 3
	AuraDummy                               AuraType = 4
	AuraModConfuse                          AuraType = 5
	AuraModCharm                            AuraType = 6
	AuraModFear                             AuraType = 7
	AuraPeriodicHeal                        AuraType = 8
	AuraModAttackSpeed                      AuraType = 9
	AuraModThreat                           AuraType = 10
	AuraModTaunt                            AuraType = 11
	AuraModStun                             AuraType = 12
	AuraModDamageDone                       AuraType = 13
	AuraModDamageTaken                      AuraType = 14
	AuraDamageShield                        AuraType = 15
	AuraModStealth                          AuraType = 16
	AuraModStealthDetect                    AuraType = 17
	AuraModInvisibility                     AuraType = 18
	AuraModInvisibilityDetect               AuraType = 19
	AuraObsModHealth                        AuraType = 20
	AuraObsModPower                         AuraType = 21
	AuraModResistance                       AuraType = 22
	AuraPeriodicTriggerSpell                AuraType = 23
	AuraPeriodicEnergize                    AuraType = 24
	AuraModPacify                           AuraType = 25
	AuraModRoot                             AuraType = 26
	AuraModSilence                          AuraType = 27
	AuraReflectSpells                       AuraType = 28
	AuraModStat                             AuraType = 29
	AuraModSkill                            AuraType = 30
	AuraModIncreaseSpeed                    AuraType = 31
	AuraModIncreaseMountedSpeed             AuraType = 32
	AuraModDecreaseSpeed                    AuraType = 33
	AuraModIncreaseHealth                   AuraType = 34
	AuraModIncreaseEnergy                   AuraType = 35
	AuraModShapeshift                       AuraType = 36
	AuraEffectImmunity                      AuraType = 37
	AuraStateImmunity                       AuraType = 38
	AuraSchoolImmunity                      AuraType = 39
	AuraDamageImmunity                      AuraType = 40
	AuraDispelImmunity                      AuraType = 41
	AuraProcTriggerSpell                    AuraType = 42
	AuraProcTriggerDamage                   AuraType = 43
	AuraTrackCreatures                      AuraType = 44
	AuraTrackResources                      AuraType = 45
	Aura46                                  AuraType = 46
	AuraModParryPercent                     AuraType = 47
	AuraPeriodicTriggerSpellFromClient      AuraType = 48
	AuraModDodgePercent                     AuraType = 49
	AuraModCriticalHealingAmount            AuraType = 50
	AuraModBlockPercent                     AuraType = 51
	AuraModWeaponCritPercent                AuraType = 52
	AuraPeriodicLeech                       AuraType = 53
	AuraModHitChance                        AuraType = 54
	AuraModSpellHitChance                   AuraType = 55
	AuraTransform                           AuraType = 56
	AuraModSpellCritChance                  AuraType = 57
	AuraModIncreaseSwimSpeed                AuraType = 58
	AuraModDamageDoneCreature               AuraType = 59
	AuraModPacifySilence                    AuraType = 60
	AuraModScale                            AuraType = 61
	AuraPeriodicHealthFunnel                AuraType = 62
	Aura63                                  AuraType = 63
	AuraPeriodicManaLeech                   AuraType = 64
	AuraModCastingSpeedNotStack             AuraType = 65
	AuraFeignDeath                          AuraType = 66
	AuraModDisarm                           AuraType = 67
	AuraModStalked                          AuraType = 68
	AuraSchoolAbsorb                        AuraType = 69
	AuraExtraAttacks                        AuraType = 70
	AuraModSpellCritChanceSchool            AuraType = 71
	AuraModPowerCostSchoolPct               AuraType = 72
	AuraModPowerCostSchool                  AuraType = 73
	AuraReflectSpellsSchool                 AuraType = 74
	AuraModLanguage                         AuraType = 75
	AuraFarSight                            AuraType = 76
	AuraMechanicImmunity                    AuraType = 77
	AuraMounted                             AuraType = 78
	AuraModDamagePercentDone                AuraType = 79
	AuraModPercentStat                      AuraType = 80
	AuraSplitDamagePct                      AuraType = 81
	AuraWaterBreathing                      AuraType = 82
	AuraModBaseResistance                   AuraType = 83
	AuraModRegen                            AuraType = 84
	AuraModPowerRegen                       AuraType = 85
	AuraChannelDeathItem                    AuraType = 86
	AuraModDamagePercentTaken               AuraType = 87
	AuraModHealthRegenPercent               AuraType = 88
	AuraPeriodicDamagePercent               AuraType = 89
	Aura90                                  AuraType = 90
	AuraModDetectRange                      AuraType = 91
	AuraPreventsFleeing                     AuraType = 92
	AuraModUnattackable                     AuraType = 93
	AuraInterruptRegen                      AuraType = 94
	AuraGhost                               AuraType = 95
	AuraSpellMagnet                         AuraType = 96
	AuraManaShield                          AuraType = 97
	AuraModSkillTalent                      AuraType = 98
	AuraModAttackPower                      AuraType = 99
	AuraAurasVisible                        AuraType = 100
	AuraModResistancePct                    AuraType = 101
	AuraModMeleeAttackPowerVersus           AuraType = 102
	AuraModTotalThreat                      AuraType = 103
	AuraWaterWalk                           AuraType = 104
	AuraFeatherFall                         AuraType = 105
	AuraHover                               AuraType = 106
	AuraAddFlatModifier                     AuraType = 107
	AuraAddPctModifier                      AuraType = 108
	AuraAddTargetTrigger                    AuraType = 109
	AuraModPowerRegenPercent                AuraType = 110
	AuraAddCasterHitTrigger                 AuraType = 111
	AuraOverrideClassScripts                AuraType = 112
	AuraModRangedDamageTaken                AuraType = 113
	AuraModRangedDamageTakenPct             AuraType = 114
	AuraModHealing                          AuraType = 115
	AuraModRegenDuringCombat                AuraType = 116
	AuraModMechanicResistance               AuraType = 117
	AuraModHealingPct                       AuraType = 118
	Aura119                                 AuraType = 119
	AuraUntrackable                         AuraType = 120
	AuraEmpathy                             AuraType = 121
	AuraModOffhandDamagePct                 AuraType = 122
	AuraModTargetResistance                 AuraType = 123
	AuraModRangedAttackPower                AuraType = 124
	AuraModMeleeDamageTaken                 AuraType = 125
	AuraModMeleeDamageTakenPct              AuraType = 126
	AuraRangedAttackPowerAttackerBonus      AuraType = 127
	AuraModPossessPet                       AuraType = 128
	AuraModSpeedAlways                      AuraType = 129
	AuraModMountedSpeedAlways               AuraType = 130
	AuraModRangedAttackPowerVersus          AuraType = 131
	AuraModIncreaseEnergyPercent            AuraType = 132
	AuraModIncreaseHealthPercent            AuraType = 133
	AuraModManaRegenInterrupt               AuraType = 134
	AuraModHealingDone                      AuraType = 135
	AuraModHealingDonePercent               AuraType = 136
	AuraModTotalStatPercentage              AuraType = 137
	AuraModMeleeHaste                       AuraType = 138
	AuraForceReaction                       AuraType = 139
	AuraModRangedHaste                      AuraType = 140
	AuraModRangedAmmoHaste                  AuraType = 141
	AuraModBaseResistancePct                AuraType = 142
	AuraModResistanceExclusive              AuraType = 143
	AuraSafeFall                            AuraType = 144
	AuraModPetTalentPoints                  AuraType = 145
	AuraAllowTamePetType                    AuraType = 146
	AuraMechanicImmunityMask                AuraType = 147
	AuraRetainComboPoints                   AuraType = 148
	AuraReducePushback                      AuraType = 149
	AuraModShieldBlockValuePct              AuraType = 150
	AuraTrackStealthed                      AuraType = 151
	AuraModDetectedRange                    AuraType = 152
	AuraSplitDamageFlat                     AuraType = 153
	AuraModStealthLevel                     AuraType = 154
	AuraModWaterBreathing                   AuraType = 155
	AuraModReputationGain                   AuraType = 156
	AuraPetDamageMulti                      AuraType = 157
	AuraModShieldBlockValue                 AuraType = 158
	AuraNoPvPCredit                         AuraType = 159
	AuraModAOEAvoidance                     AuraType = 160
	AuraModHealthRegenInCombat              AuraType = 161
	AuraPowerBurn                           AuraType = 162
	AuraModCritDamageBonus                  AuraType = 163
	Aura164                                 AuraType = 164
	AuraMeleeAttackPowerAttackerBonus       AuraType = 165
	AuraModAttackPowerPct                   AuraType = 166
	AuraModRangedAttackPowerPct             AuraType = 167
	AuraModDamageDoneVersus                 AuraType = 168
	AuraModCritPercentVersus                AuraType = 169
	AuraDetectAmore                         AuraType = 170
	AuraModSpeedNotStack                    AuraType = 171
	AuraModMountedSpeedNotStack             AuraType = 172
	Aura173                                 AuraType = 173
	AuraModSpellDamageOfStatPercent         AuraType = 174
	AuraModSpellHealingOfStatPercent        AuraType = 175
	AuraSpiritOfRedemption                  AuraType = 176
	AuraAoeCharm                            AuraType = 177
	AuraModDebuffResistance                 AuraType = 178
	AuraModAttackerSpellCritChance          AuraType = 179
	AuraModFlatSpellDamageVersus            AuraType = 180
	Aura181                                 AuraType = 181
	AuraModResistanceOfStatPercent          AuraType = 182
	AuraModCriticalThreat                   AuraType = 183
	AuraModAttackerMeleeHitChance           AuraType = 184
	AuraModAttackerRangedHitChance          AuraType = 185
	AuraModAttackerSpellHitChance           AuraType = 186
	AuraModAttackerMeleeCritChance          AuraType = 187
	AuraModAttackerRangedCritChance         AuraType = 188
	AuraModRating                           AuraType = 189
	AuraModFactionReputationGain            AuraType = 190
	AuraUseNormalMovementSpeed              AuraType = 191
	AuraModMeleeRangedHaste                 AuraType = 192
	AuraMeleeSlow                           AuraType = 193
	AuraModTargetAbsorbSchool               AuraType = 194
	AuraModTargetAbilityAbsorbSchool        AuraType = 195
	AuraModCooldown                         AuraType = 196
	AuraModAttackerSpellAndWeaponCritChance AuraType = 197
	Aura198                                 AuraType = 198
	AuraModIncreasesSpellPctToHit           AuraType = 199
	AuraModXpPct                            AuraType = 200
	AuraFly                                 AuraType = 201
	AuraIgnoreCombatResult                  AuraType = 202
	AuraModAttackerMeleeCritDamage          AuraType = 203
	AuraModAttackerRangedCritDamage         AuraType = 204
	AuraModSchoolCritDmgTaken               AuraType = 205
	AuraModIncreaseFlightSpeed              AuraType = 206
	AuraModIncreaseMountedFlightSpeed       AuraType = 207
	AuraModFlightSpeedAlways                AuraType = 208
	AuraModMountedFlightSpeedAlways         AuraType = 209
	AuraModFlightSpeedNotStacking           AuraType = 210
	AuraModFlightSpeedMountedNotStacking    AuraType = 211
	AuraModRangedAttackPowerOfStatPercent   AuraType = 212
	AuraModRageFromDamageDealt              AuraType = 213
	Aura214                                 AuraType = 214
	ArenaPreparation                        AuraType = 215
	AuraHasteSpells                         AuraType = 216
	AuraModMeleeHaste2                      AuraType = 217
	AuraHasteRanged                         AuraType = 218
	AuraModManaRegenFromStat                AuraType = 219
	AuraModRatingFromStat                   AuraType = 220
	AuraModDetaunt                          AuraType = 221
	Aura222                                 AuraType = 222
	AuraRaidProcFromCharge                  AuraType = 223
	Aura224                                 AuraType = 224
	AuraRaidProcFromChargeWithValue         AuraType = 225
	AuraPeriodicDummy                       AuraType = 226
	AuraPeriodicTriggerSpellWithValue       AuraType = 227
	AuraDetectStealth                       AuraType = 228
	AuraModAOEDamageAvoidance               AuraType = 229
	Aura230                                 AuraType = 230
	AuraProcTriggerSpellWithValue           AuraType = 231
	AuraMechanicDurationMod                 AuraType = 232
	AuraChangeModelForAllHumanoids          AuraType = 233
	AuraMechanicDurationModNotStack         AuraType = 234
	AuraModDispelResist                     AuraType = 235
	AuraControlVehicle                      AuraType = 236
	AuraModSpellDamageOfAttackPower         AuraType = 237
	AuraModSpellHealingOfAttackPower        AuraType = 238
	AuraModScale2                           AuraType = 239
	AuraModExpertise                        AuraType = 240
	AuraForceMoveForward                    AuraType = 241
	AuraModSpellDamageFromHealing           AuraType = 242
	AuraModFaction                          AuraType = 243
	AuraComprehendLanguage                  AuraType = 244
	AuraModAuraDurationByDispel             AuraType = 245
	AuraModAuraDurationByDispelNotStack     AuraType = 246
	AuraCloneCaster                         AuraType = 247
	AuraModCombatResultChance               AuraType = 248
	AuraConvertRune                         AuraType = 249
	AuraModIncreaseHealth2                  AuraType = 250
	AuraModEnemyDodge                       AuraType = 251
	AuraModSpeedSlowAll                     AuraType = 252
	AuraModBlockCritChance                  AuraType = 253
	AuraModDisarmOffhand                    AuraType = 254
	AuraModMechanicDamageTakenPercent       AuraType = 255
	AuraNoReagentUse                        AuraType = 256
	AuraModTargetResistBySpellClass         AuraType = 257
	Aura258                                 AuraType = 258
	AuraModHotPct                           AuraType = 259
	AuraScreenEffect                        AuraType = 260
	AuraPhase                               AuraType = 261
	AuraAbilityIgnoreAurastate              AuraType = 262
	AuraAllowOnlyAbility                    AuraType = 263
	Aura264                                 AuraType = 264
	Aura265                                 AuraType = 265
	Aura266                                 AuraType = 266
	AuraModImmuneAuraApplySchool            AuraType = 267
	AuraModAttackPowerOfStatPercent         AuraType = 268
	AuraModIgnoreTargetResistModifiers      AuraType = 269
	AuraModAbilityIgnoreTargetResist        AuraType = 270
	AuraModDamageFromCaster                 AuraType = 271
	AuraIgnoreMeleeReset                    AuraType = 272
	AuraXRay                                AuraType = 273
	AuraAbilityConsumeNoAmmo                AuraType = 274
	AuraModIgnoreShapeshift                 AuraType = 275
	AuraModDamageDoneForMechanic            AuraType = 276
	AuraModMaxAffectedTargets               AuraType = 277
	AuraModDisarmRanged                     AuraType = 278
	AuraInitializeImages                    AuraType = 279
	AuraModArmorPenetrationPct              AuraType = 280
	AuraModHonorGainPct                     AuraType = 281
	AuraModBaseHealthPct                    AuraType = 282
	AuraModHealingReceived                  AuraType = 283
	AuraLinked                              AuraType = 284
	AuraModAttackPowerOfArmor               AuraType = 285
	AuraAbilityPeriodicCrit                 AuraType = 286
	AuraDeflectSpells                       AuraType = 287
	AuraIgnoreHitDirection                  AuraType = 288
	AuraPreventDurabilityLoss               AuraType = 289
	AuraModCritPct                          AuraType = 290
	AuraModXpQuestPct                       AuraType = 291
	AuraOpenStable                          AuraType = 292
	AuraOverrideSpells                      AuraType = 293
	AuraPreventRegeneratePower              AuraType = 294
	Aura295                                 AuraType = 295
	AuraSetVehicleId                        AuraType = 296
	AuraBlockSpellFamily                    AuraType = 297
	AuraStrangulate                         AuraType = 298
	Aura299                                 AuraType = 299
	AuraShareDamagePct                      AuraType = 300
	AuraSchoolHealAbsorb                    AuraType = 301
	Aura302                                 AuraType = 302
	AuraModDamageDoneVersusAurastate        AuraType = 303
	AuraModFakeInebriate                    AuraType = 304
	AuraModMinimumSpeed                     AuraType = 305
	Aura306                                 AuraType = 306
	AuraHealAbsorbTest                      AuraType = 307
	AuraModCritChanceForCaster              AuraType = 308
	Aura309                                 AuraType = 309
	AuraModCreatureAOEDamageAvoidance       AuraType = 310
	Aura311                                 AuraType = 311
	Aura312                                 AuraType = 312
	Aura313                                 AuraType = 313
	AuraPreventResurrection                 AuraType = 314
	AuraUnderwaterWalking                   AuraType = 315
	AuraPeriodicHaste                       AuraType = 316
	TotalAuras                              AuraType = 317
)

// SpellFamily names from AC.
type SpellFamily uint32

const (
	SpellFamilyGeneric     SpellFamily = 0
	SpellFamilyWarrior     SpellFamily = 4
	SpellFamilyPaladin     SpellFamily = 10
	SpellFamilyHunter      SpellFamily = 9
	SpellFamilyRogue       SpellFamily = 8
	SpellFamilyPriest      SpellFamily = 6
	SpellFamilyDeathKnight SpellFamily = 15
	SpellFamilyShaman      SpellFamily = 11
	SpellFamilyMage        SpellFamily = 3
	SpellFamilyWarlock     SpellFamily = 5
	SpellFamilyDruid       SpellFamily = 11
)

// InterruptFlags from AC.
const (
	SpellInterruptFlagNone           uint32 = 0x00000000
	SpellInterruptFlagCasting        uint32 = 0x00000001
	SpellInterruptFlagMovement       uint32 = 0x00000002
	SpellInterruptFlagDamage         uint32 = 0x00000004
	SpellInterruptFlagDamageWithStun uint32 = 0x00000008
	SpellInterruptFlagAutoAttack     uint32 = 0x00000010
	SpellInterruptFlagAIButNotPlayer uint32 = 0x00000020
	SpellInterruptFlagJump           uint32 = 0x00000040
	SpellInterruptFlagNotUnderwater  uint32 = 0x00000080
	SpellInterruptFlagMeleeAttack    uint32 = 0x00000100
)

// ProcFlags from AC.
const (
	ProcFlagNone                    uint32 = 0x00000000
	ProcFlagSuccessfulCast          uint32 = 0x00000001
	ProcFlagTakeDamage              uint32 = 0x00000002
	ProcFlagOnKill                  uint32 = 0x00000004
	ProcFlagDeath                   uint32 = 0x00000008
	ProcFlagSuccessfulMeleeHit      uint32 = 0x00000010
	ProcFlagOnMeleeVictim           uint32 = 0x00000020
	ProcFlagOnAutoAttackVictim      uint32 = 0x00000040
	ProcFlagOnAutoAttackKill        uint32 = 0x00000080
	ProcFlagOnSpellHit              uint32 = 0x00000100
	ProcFlagOnSpellVictim           uint32 = 0x00000200
	ProcFlagOnPeriodic              uint32 = 0x00000400
	ProcFlagOnPeriodicVictim        uint32 = 0x00000800
	ProcFlagOnMeleeCrit             uint32 = 0x00001000
	ProcFlagOnSpellCrit             uint32 = 0x00002000
	ProcFlagOnPeriodicCrit          uint32 = 0x00004000
	ProcFlagOnRemove                uint32 = 0x00008000
	ProcFlagOnKillVictim            uint32 = 0x00010000
	ProcFlagOnMeleeTakenCrit        uint32 = 0x00020000
	ProcFlagOnSpellTakenHit         uint32 = 0x00040000
	ProcFlagOnAutoAttackTakenCrit   uint32 = 0x00080000
	ProcFlagOnPeriodicTakenCrit     uint32 = 0x00100000
	ProcFlagOnTakenPeriodic         uint32 = 0x00200000
	ProcFlagOnTakenMelee            uint32 = 0x00400000
	ProcFlagOnTakenAutoAttack       uint32 = 0x00800000
	ProcFlagOnTakenDamage           uint32 = 0x01000000
	ProcFlagOnTrapActivation        uint32 = 0x02000000
	ProcFlagOnAutoAttackCritVictim  uint32 = 0x04000000
	ProcFlagOnAutoAttackBlockVictim uint32 = 0x08000000
)

// Spell attributes flags (most commonly used).
const (
	SpellAttr0CantCrit       uint32 = 0x00000080
	SpellAttr0OnNextSwing    uint32 = 0x00000040
	SpellAttr0IsAbility      uint32 = 0x00000100
	SpellAttr0IsTradeskill   uint32 = 0x00000200
	SpellAttr0Passive        uint32 = 0x00000400
	SpellAttr0DoNotDisplay   uint32 = 0x00000080
	SpellAttr0NoImmunities   uint32 = 0x00000020
	SpellAttr0NoStealBenefit uint32 = 0x00040000
	SpellAttr0NoParry        uint32 = 0x00000010
	SpellAttr0NoCrit         uint32 = 0x02000000
	// SPELL_ATTR0_NOT_IN_COMBAT_ONLY_PEACEFUL — cannot be used in combat.
	SpellAttr0NotInCombatOnlyPeaceful uint32 = 0x10000000
)

// SpellAttrEx flags.
const (
	SpellAttrExNegative              uint32 = 0x00000004
	SpellAttrExNoStackWithBeneficial uint32 = 0x00000100
	SpellAttrExChanneled1            uint32 = 0x00000020
	SpellAttrExChanneled2            uint32 = 0x00000040
	SpellAttrExAllowWhileStealthed   uint32 = 0x00000800
	SpellAttrExUseAllMana            uint32 = 0x00002000
	SpellAttrExExcludeCaster         uint32 = 0x00000008
	SpellAttrExFinishingMoveDamage   uint32 = 0x00200000
	SpellAttrExFinishingMoveDuration uint32 = 0x00400000
)

// SpellAttrEx2 flags.
const (
	SpellAttr2AutoRepeat                uint32 = 0x00000002
	SpellAttr2CantCrit                  uint32 = 0x00000800
	SpellAttr2AllowWhileNotShapeshifted uint32 = 0x00000010
)

// SpellDamageClass from AC.
const (
	SpellDamageClassNone   uint32 = 0
	SpellDamageClassMagic  uint32 = 0
	SpellDamageClassMelee  uint32 = 1
	SpellDamageClassRanged uint32 = 2
)

// SpellPreventionType from AC.
const (
	SpellPreventionNone    uint32 = 0
	SpellPreventionSilence uint32 = 1
	SpellPreventionPacify  uint32 = 2
)

// SpellCastResult defines the result of a spell cast attempt.
type SpellCastResult uint32

const (
	SpellCastSuccess                SpellCastResult = 0
	SpellCastFailedSpellFailed      SpellCastResult = 6
	SpellCastNoMana                 SpellCastResult = 12
	SpellCastTargetTooFar           SpellCastResult = 51
	SpellCastAlreadyActive          SpellCastResult = 10
	SpellCastCantDoThatYet          SpellCastResult = 165
	SpellCastFailedSpellInterrupted SpellCastResult = 14
	SpellCastFailedTargetAurastate  SpellCastResult = 39
	SpellCastFailedCasterAurastate  SpellCastResult = 40
	SpellCastFailedCasterIsDead     SpellCastResult = 69
	SpellCastFailedTargetIsDead     SpellCastResult = 70
	SpellCastFailedUnknownSpell     SpellCastResult = 171
	SpellCastFailedSpellOnCooldown  SpellCastResult = 75
)

// SpellState defines the state of a spell cast.
type SpellState uint32

const (
	SpellStateNone     SpellState = 0
	SpellStateCasting  SpellState = 1
	SpellStateCastTime SpellState = 2
	SpellStateActive   SpellState = 3
	SpellStateFinished SpellState = 4
)

// SpellTargetType defines who the spell can target.
type SpellTargetType uint32

const (
	SpellTargetNone     SpellTargetType = 0
	SpellTargetUnit     SpellTargetType = 1
	SpellTargetEnemy    SpellTargetType = 2
	SpellTargetFriendly SpellTargetType = 3
	SpellTargetSelf     SpellTargetType = 4
	SpellTargetArea     SpellTargetType = 5
	SpellTargetAll      SpellTargetType = 6
)

// PowerType for spell cost.
type SpellPowerType uint32

const (
	SpellPowerMana      SpellPowerType = 0
	SpellPowerRage      SpellPowerType = 1
	SpellPowerFocus     SpellPowerType = 2
	SpellPowerEnergy    SpellPowerType = 3
	SpellPowerHappiness SpellPowerType = 4
)

// SpellEffectIndex for effect slots.
type SpellEffectIndex int

const (
	SpellEffectIndex0 SpellEffectIndex = 0
	SpellEffectIndex1 SpellEffectIndex = 1
	SpellEffectIndex2 SpellEffectIndex = 2
)

// SpellCustomAttributes from AC (computed by SpellMgr, not in DBC).
const (
	SpellAttr0CuEnchantProc                uint32 = 0x00000001
	SpellAttr0CuConeBack                   uint32 = 0x00000002
	SpellAttr0CuConeLine                   uint32 = 0x00000004
	SpellAttr0CuShareDamage                uint32 = 0x00000008
	SpellAttr0CuNoInitialThreat            uint32 = 0x00000010
	SpellAttr0CuAuraCc                     uint32 = 0x00000020
	SpellAttr0CuDontBreakStealth           uint32 = 0x00000040
	SpellAttr0CuNoPvpFlag                  uint32 = 0x00000080
	SpellAttr0CuDirectDamage               uint32 = 0x00000100
	SpellAttr0CuCharge                     uint32 = 0x00000200
	SpellAttr0CuPickpocket                 uint32 = 0x00000400
	SpellAttr0CuIgnoreEvasion              uint32 = 0x00000800
	SpellAttr0CuNegativeEff0               uint32 = 0x00001000
	SpellAttr0CuNegativeEff1               uint32 = 0x00002000
	SpellAttr0CuNegativeEff2               uint32 = 0x00004000
	SpellAttr0CuIgnoreArmor                uint32 = 0x00008000
	SpellAttr0CuReqTargetFacingCaster      uint32 = 0x00010000
	SpellAttr0CuReqCasterBehindTarget      uint32 = 0x00020000
	SpellAttr0CuAllowInflightTarget        uint32 = 0x00040000
	SpellAttr0CuNeedsAmmoData              uint32 = 0x00080000
	SpellAttr0CuBinarySpell                uint32 = 0x00100000
	SpellAttr0CuNoPositiveTakenBonus       uint32 = 0x00200000
	SpellAttr0CuSingleAuraStack            uint32 = 0x00400000
	SpellAttr0CuSchoolmaskNormalWithMagic  uint32 = 0x00800000
	SpellAttr0CuAuraCannotBeSaved          uint32 = 0x01000000
	SpellAttr0CuPositiveEff0               uint32 = 0x02000000
	SpellAttr0CuPositiveEff1               uint32 = 0x04000000
	SpellAttr0CuPositiveEff2               uint32 = 0x08000000
	SpellAttr0CuForceSendCategoryCooldowns uint32 = 0x10000000
	SpellAttr0CuForceAuraSaving            uint32 = 0x20000800
	SpellAttr0CuOnlyOneAreaAura            uint32 = 0x20000000
	SpellAttr0CuEncounterReward            uint32 = 0x40000000
	SpellAttr0CuBypassMechanicImmunity     uint32 = 0x80000000

	SpellAttr0CuNegative = SpellAttr0CuNegativeEff0 | SpellAttr0CuNegativeEff1 | SpellAttr0CuNegativeEff2
	SpellAttr0CuPositive = SpellAttr0CuPositiveEff0 | SpellAttr0CuPositiveEff1 | SpellAttr0CuPositiveEff2
)

// TriggerCastFlags controls how a spell is triggered (AC TriggerCastFlags).
type TriggerCastFlags uint32

const (
	TriggeredNone                     TriggerCastFlags = 0x00000000
	TriggeredIgnorePowerCostReagents  TriggerCastFlags = 0x00000001
	TriggeredIgnoreSpellCost          TriggerCastFlags = 0x00000002
	TriggeredIgnoreGroupCooldown      TriggerCastFlags = 0x00000004
	TriggeredIgnoreCooldowns          TriggerCastFlags = 0x00000008
	TriggeredIgnoreCastInFlight       TriggerCastFlags = 0x00000010
	TriggeredIgnoreCastInProgress     TriggerCastFlags = 0x00000080
	TriggeredCastDirectly             TriggerCastFlags = 0x00000200
	TriggeredTriggered                TriggerCastFlags = 0x00000400
	TriggeredIgnoreAuraInterruptFlags TriggerCastFlags = 0x00000800
	TriggeredIgnoreGCD                TriggerCastFlags = 0x00001000
	TriggeredFullMask                 TriggerCastFlags = 0x00003FFF
)
