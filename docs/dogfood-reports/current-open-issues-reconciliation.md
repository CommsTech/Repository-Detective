# Current open issues reconciliation — commstech/Repository-Detective

Generated: 2026-06-06 22:26 UTC

Reconciled against scan **`a8bb4cddd72ab80c`** (1101 finding instances).

## Summary

| Metric | Count |
|--------|------:|
| Gitea open issues exported | 288 |
| active_present_in_latest_scan | 41 |
| duplicate_existing_fingerprint | 69 |
| missing_local_mapping_backfilled | 4 |
| needs_human_review | 2 |
| out_of_scope_for_current_batch | 29 |
| resolved_absent_from_latest_scan | 143 |

## Detail

| issue | title | fingerprint | source | severity | scan presence | classification | canonical | action | evidence |
|------:|-------|-------------|--------|----------|---------------|----------------|-----------|--------|----------|
| #48 | Ops: homelab AI/Qdrant connectivity from Docker | `` |  |  | unknown | needs_human_review |  | review_untracked | unverified |
| #49 | Ops: Docker Trivy install when GitHub CDN blocked | `` |  |  | unknown | needs_human_review |  | review_untracked | unverified |
| #50 | [LOW] Possible disconnected code path: file not re | `rd-c2dc9d7d729d2936` | graph (AI auditor) | low | absent | resolved_absent_from_latest_scan | 50 | evidence_closure | pending_verify |
| #51 | [MEDIUM] Possible suspicious code island — disconn | `rd-7961138ac5f3b655` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 51 | evidence_closure | pending_verify |
| #52 | [LOW] Possible disconnected code path: file not re | `rd-fce78c673d11c86a` | graph (AI auditor) | low | absent | resolved_absent_from_latest_scan | 52 | evidence_closure | pending_verify |
| #53 | [MEDIUM] Possible internal infrastructure referenc | `rd-9a6eaeee6abc7aa7` | static | medium | present | active_present_in_latest_scan | 53 | fix_in_code_batch | unverified |
| #54 | [MEDIUM] Potentially disconnected package/module — | `rd-467da43f25ecc7f3` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #55 | [MEDIUM] Possible suspicious code island — disconn | `rd-7d9e0d6673b53127` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 55 | evidence_closure | pending_verify |
| #56 | [MEDIUM] Semgrep finding: go.lang.security.audit.c | `rd-03bb1f30a7adb71a` | semgrep | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #57 | [MEDIUM] Potentially disconnected package/module — | `rd-72480e3af4ba5760` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #58 | [MEDIUM] Potentially disconnected package/module — | `rd-84283703a42c65a0` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 58 | evidence_closure | pending_verify |
| #59 | [MEDIUM] Possible suspicious code island — disconn | `rd-5edd9a11ba7f15f8` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 59 | evidence_closure | pending_verify |
| #60 | [MEDIUM] Potential reliability issue: ignored erro | `rd-6e1463f2206db985` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #61 | [MEDIUM] Potential reliability issue: ignored erro | `rd-5db7a89fa16b80ea` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 61 | evidence_closure | pending_verify |
| #62 | [MEDIUM] Potential reliability issue: ignored erro | `rd-14e31d9224b6f115` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #63 | [MEDIUM] Potential reliability issue: ignored erro | `rd-d60e415828af3d1d` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #64 | [MEDIUM] Potentially disconnected package/module — | `rd-55401f16f5afef68` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #65 | [MEDIUM] Possible suspicious code island — disconn | `rd-729fb4cf14eae1a2` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 65 | evidence_closure | pending_verify |
| #66 | [MEDIUM] Possible internal infrastructure referenc | `rd-b67022bb59a95c9b` | static | medium | present | active_present_in_latest_scan | 66 | fix_in_code_batch | unverified |
| #67 | [MEDIUM] Potentially disconnected package/module — | `rd-16afc540e7ef15e4` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 67 | evidence_closure | pending_verify |
| #68 | [MEDIUM] Possible suspicious code island — disconn | `rd-16967ae45443be99` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 68 | evidence_closure | pending_verify |
| #69 | [MEDIUM] Potentially disconnected package/module — | `rd-9d287e184bcf550b` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #70 | [MEDIUM] Possible suspicious code island — disconn | `rd-6a4d530a818edbc9` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 70 | evidence_closure | pending_verify |
| #71 | [MEDIUM] Potentially disconnected package/module — | `rd-1adc085fe046f83d` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #72 | [LOW] Possible disconnected code path: file not re | `rd-7315f2481cb1b0b4` | graph (AI auditor) | low | absent | resolved_absent_from_latest_scan | 72 | evidence_closure | pending_verify |
| #73 | [MEDIUM] Potentially disconnected package/module — | `rd-9546ecc941c4d733` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #74 | [MEDIUM] Potentially disconnected package/module — | `rd-dde2e2ecf734349a` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #75 | [MEDIUM] Possible suspicious code island — disconn | `rd-54c67d8db86c32d2` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 75 | evidence_closure | pending_verify |
| #76 | [MEDIUM] Possible suspicious code island — disconn | `rd-5d7eb7dc9ba34132` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #77 | [MEDIUM] Potential reliability issue: panic in lib | `rd-ee799443f5dc37c5` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #78 | [MEDIUM] Potentially disconnected package/module — | `rd-ac66459d62a77aef` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #79 | [MEDIUM] Possible suspicious code island — disconn | `rd-55cf2dec27c7380b` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 79 | evidence_closure | pending_verify |
| #80 | [MEDIUM] Potentially disconnected package/module — | `rd-8a46979f3a28183a` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 80 | evidence_closure | pending_verify |
| #81 | [MEDIUM] Potentially disconnected package/module — | `rd-8fec2eee8721a604` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 81 | evidence_closure | pending_verify |
| #82 | [MEDIUM] Possible suspicious code island — disconn | `rd-999ad0b7228342bd` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 82 | evidence_closure | pending_verify |
| #83 | [MEDIUM] Potential reliability issue: ignored erro | `rd-280d0f0fe459669f` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 83 | evidence_closure | pending_verify |
| #84 | [MEDIUM] Potentially disconnected package/module — | `rd-762f59ef98441dd1` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 84 | evidence_closure | pending_verify |
| #85 | [MEDIUM] Potentially disconnected package/module — | `rd-fcc59bc710efe03b` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #86 | [MEDIUM] Potential reliability issue: ignored erro | `rd-6d38bc4d3577c0d6` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 86 | evidence_closure | pending_verify |
| #87 | [MEDIUM] Possible suspicious code island — disconn | `rd-d96ec81e239cb486` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #88 | [MEDIUM] Potential reliability issue: ignored erro | `rd-a19cfe26c27e1ecb` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #89 | [MEDIUM] Potentially disconnected package/module — | `rd-a4104968550edb1c` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 89 | evidence_closure | pending_verify |
| #90 | [MEDIUM] Potential reliability issue: ignored erro | `rd-80bd0c20a2fa9d1f` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 90 | evidence_closure | pending_verify |
| #91 | [MEDIUM] Potential reliability issue: ignored erro | `rd-11c7e16270f942db` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 91 | evidence_closure | pending_verify |
| #92 | [MEDIUM] Potential reliability issue: ignored erro | `rd-567499d75f8db670` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 92 | evidence_closure | pending_verify |
| #93 | [MEDIUM] Potentially disconnected package/module — | `rd-c4eb4cf387d44617` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #94 | [MEDIUM] Potentially disconnected package/module — | `rd-a80ad5b7f9fd1b24` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #95 | [MEDIUM] Possible suspicious code island — disconn | `rd-e447a696b381116c` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #96 | [MEDIUM] Potentially disconnected package/module — | `rd-7bd5fd04eb7c59e6` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #97 | [MEDIUM] Potentially disconnected package/module — | `rd-d669030cc1a762df` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #98 | [MEDIUM] Potentially disconnected package/module — | `rd-d608271f97300c5f` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 98 | evidence_closure | pending_verify |
| #99 | [MEDIUM] Possible suspicious code island — disconn | `rd-02883232e7265639` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #100 | Code Review Summary - 64 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #101 | [LOW] Possible disconnected code path: file not re | `rd-c2dc9d7d729d2936` | graph (AI auditor) | low | absent | duplicate_existing_fingerprint | 50 | link_canonical | pending_verify |
| #102 | [MEDIUM] Possible suspicious code island — disconn | `rd-7961138ac5f3b655` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 51 | link_canonical | pending_verify |
| #103 | [LOW] Possible disconnected code path: file not re | `rd-fce78c673d11c86a` | graph (AI auditor) | low | absent | duplicate_existing_fingerprint | 52 | link_canonical | pending_verify |
| #104 | [MEDIUM] Possible internal infrastructure referenc | `rd-9a6eaeee6abc7aa7` | static | medium | present | duplicate_existing_fingerprint | 53 | link_canonical | unverified |
| #105 | [MEDIUM] Possible suspicious code island — disconn | `rd-455cff459bac20c2` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 105 | evidence_closure | pending_verify |
| #106 | [MEDIUM] Potentially disconnected package/module — | `rd-249d77ae9bd4a305` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #107 | [MEDIUM] Possible suspicious code island — disconn | `rd-7d9e0d6673b53127` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 55 | link_canonical | pending_verify |
| #108 | [MEDIUM] Potentially disconnected package/module — | `rd-f6385c70b6bdc80a` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #109 | [MEDIUM] Potentially disconnected package/module — | `rd-1d9de7dfbf1983bb` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #110 | [MEDIUM] Potentially disconnected package/module — | `rd-6a6504b586da04da` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #111 | [MEDIUM] Possible suspicious code island — disconn | `rd-5edd9a11ba7f15f8` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 59 | link_canonical | pending_verify |
| #112 | [MEDIUM] Potential reliability issue: ignored erro | `rd-5db7a89fa16b80ea` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 61 | link_canonical | pending_verify |
| #113 | [MEDIUM] Potentially disconnected package/module — | `rd-5f37593a0fe4dbbb` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #114 | [MEDIUM] Potentially disconnected package/module — | `rd-779f6ba996304cc6` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #115 | [MEDIUM] Potentially disconnected package/module — | `rd-e8700b65a4f04552` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #116 | [MEDIUM] Potentially disconnected package/module — | `rd-d83beee3abaeaecb` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #117 | [MEDIUM] Potentially disconnected package/module — | `rd-4a8b0d8577647376` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #118 | [MEDIUM] Potentially disconnected package/module — | `rd-a8347903413814b6` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #119 | [MEDIUM] Potentially disconnected package/module — | `rd-5a686df8c4112b0f` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #120 | [MEDIUM] Potential reliability issue: ignored erro | `rd-14ab8e7406280f9e` | reliability (AI auditor) | medium | present | active_present_in_latest_scan | 120 | fix_in_code_batch | unverified |
| #121 | [MEDIUM] Potential reliability issue: ignored erro | `rd-1a163256ff13cf90` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 121 | evidence_closure | pending_verify |
| #122 | [MEDIUM] Potential reliability issue: ignored erro | `rd-0170bee0566f8e87` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 122 | evidence_closure | pending_verify |
| #123 | [MEDIUM] Potential reliability issue: ignored erro | `rd-f4baf545376d3fe4` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 123 | evidence_closure | pending_verify |
| #124 | [MEDIUM] Potential reliability issue: ignored erro | `rd-a979086a6e89029f` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 124 | evidence_closure | pending_verify |
| #125 | [MEDIUM] Potentially disconnected package/module — | `rd-021a191dccab877d` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 125 | evidence_closure | pending_verify |
| #126 | [MEDIUM] Potentially disconnected package/module — | `rd-c041190766428f7e` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #127 | [MEDIUM] Possible suspicious code island — disconn | `rd-f8816f9138531457` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 127 | evidence_closure | pending_verify |
| #128 | [MEDIUM] Potential reliability issue: ignored erro | `rd-d0a05cf6594a828f` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 128 | evidence_closure | pending_verify |
| #129 | [MEDIUM] Possible suspicious code island — disconn | `rd-afecd70388e2150d` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #130 | [MEDIUM] Potential reliability issue: ignored erro | `rd-17ce603a440b2a8a` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 130 | evidence_closure | pending_verify |
| #131 | [MEDIUM] Potential reliability issue: ignored erro | `rd-d391ab02bcc6f715` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 131 | evidence_closure | pending_verify |
| #132 | [MEDIUM] Potentially disconnected package/module — | `rd-fa7e110fdbb2d915` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #133 | [MEDIUM] Potentially disconnected package/module — | `rd-6fee9a7d8511769f` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #134 | [MEDIUM] Potential reliability issue: ignored erro | `rd-01184640a65833ca` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 134 | evidence_closure | pending_verify |
| #135 | [MEDIUM] Possible suspicious code island — disconn | `rd-5f4038fdf131f585` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #136 | [MEDIUM] Potential reliability issue: ignored erro | `rd-52ca71ddebab9092` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 136 | evidence_closure | pending_verify |
| #137 | [MEDIUM] Potential reliability issue: ignored erro | `rd-89bda9c93b10f19b` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 137 | evidence_closure | pending_verify |
| #138 | [MEDIUM] Potentially disconnected package/module — | `rd-bb086a515a24a6e1` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #139 | [MEDIUM] Possible suspicious code island — disconn | `rd-d4eec8ece4ba6542` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #140 | [MEDIUM] Potential reliability issue: ignored erro | `rd-8a66eeb95c8d98f0` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 140 | evidence_closure | pending_verify |
| #141 | [MEDIUM] Potential reliability issue: ignored erro | `rd-155765bc2cb8e288` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 141 | evidence_closure | pending_verify |
| #142 | [MEDIUM] Possible suspicious code island — disconn | `rd-257a10f8b201fb9c` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #143 | [MEDIUM] Possible internal infrastructure referenc | `rd-bc82928a9cc02518` | static | medium | present | active_present_in_latest_scan | 143 | fix_in_code_batch | unverified |
| #144 | [MEDIUM] Possible internal infrastructure referenc | `rd-c073c4b84ab723fb` | static | medium | present | active_present_in_latest_scan | 144 | fix_in_code_batch | unverified |
| #145 | [MEDIUM] Possible internal infrastructure referenc | `rd-10d09a1086712459` | static | medium | present | active_present_in_latest_scan | 145 | fix_in_code_batch | unverified |
| #146 | [MEDIUM] Potentially disconnected package/module — | `rd-d0f89a5ed3eff4de` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #147 | [MEDIUM] Potentially disconnected package/module — | `rd-19d3115a19de280c` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #148 | [MEDIUM] Potentially disconnected package/module — | `rd-f90e95004ce32641` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #149 | [MEDIUM] Possible suspicious code island — disconn | `rd-154940a456812e32` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #150 | [MEDIUM] Potential reliability issue: ignored erro | `rd-4aad6eddd56a9488` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan | 150 | evidence_closure | pending_verify |
| #151 | Code Review Summary - 77 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #152 | [LOW] Possible disconnected code path: file not re | `rd-c2dc9d7d729d2936` | graph (AI auditor) | low | absent | duplicate_existing_fingerprint | 50 | link_canonical | pending_verify |
| #153 | [MEDIUM] Possible suspicious code island — disconn | `rd-7961138ac5f3b655` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 51 | link_canonical | pending_verify |
| #154 | [LOW] Possible disconnected code path: file not re | `rd-fce78c673d11c86a` | graph (AI auditor) | low | absent | duplicate_existing_fingerprint | 52 | link_canonical | pending_verify |
| #155 | [MEDIUM] Possible internal infrastructure referenc | `rd-9a6eaeee6abc7aa7` | static | medium | present | duplicate_existing_fingerprint | 53 | link_canonical | unverified |
| #156 | [MEDIUM] Possible suspicious code island — disconn | `rd-455cff459bac20c2` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 105 | link_canonical | pending_verify |
| #157 | [MEDIUM] Potentially disconnected package/module — | `rd-c4265df7cf7a4b3b` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #158 | [MEDIUM] Possible suspicious code island — disconn | `rd-7d9e0d6673b53127` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 55 | link_canonical | pending_verify |
| #159 | [MEDIUM] Potentially disconnected package/module — | `rd-b253e9ef7e0c17cd` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #160 | [MEDIUM] Potentially disconnected package/module — | `rd-84283703a42c65a0` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 58 | link_canonical | pending_verify |
| #161 | [MEDIUM] Potentially disconnected package/module — | `rd-84bb833d16486b89` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #162 | [MEDIUM] Possible suspicious code island — disconn | `rd-729fb4cf14eae1a2` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 65 | link_canonical | pending_verify |
| #163 | [MEDIUM] Possible internal infrastructure referenc | `rd-b67022bb59a95c9b` | static | medium | present | duplicate_existing_fingerprint | 66 | link_canonical | unverified |
| #164 | [MEDIUM] Potentially disconnected package/module — | `rd-16afc540e7ef15e4` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 67 | link_canonical | pending_verify |
| #165 | [MEDIUM] Potentially disconnected package/module — | `rd-159d68dbc7f4197b` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #166 | [MEDIUM] Possible suspicious code island — disconn | `rd-16967ae45443be99` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 68 | link_canonical | pending_verify |
| #167 | [MEDIUM] Possible suspicious code island — disconn | `rd-6a4d530a818edbc9` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 70 | link_canonical | pending_verify |
| #168 | [MEDIUM] Potentially disconnected package/module — | `rd-7be9f0223da4f7d6` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #169 | [LOW] Possible disconnected code path: file not re | `rd-7315f2481cb1b0b4` | graph (AI auditor) | low | absent | duplicate_existing_fingerprint | 72 | link_canonical | pending_verify |
| #170 | [MEDIUM] Potentially disconnected package/module — | `rd-b3738ec9a8066c01` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #171 | [MEDIUM] Potentially disconnected package/module — | `rd-bdc92cd443f81c94` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #172 | [MEDIUM] Possible suspicious code island — disconn | `rd-54c67d8db86c32d2` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 75 | link_canonical | pending_verify |
| #173 | [MEDIUM] Potentially disconnected package/module — | `rd-9d06fdda23ba50ac` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #174 | [MEDIUM] Possible suspicious code island — disconn | `rd-55cf2dec27c7380b` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 79 | link_canonical | pending_verify |
| #175 | [MEDIUM] Potentially disconnected package/module — | `rd-8a46979f3a28183a` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 80 | link_canonical | pending_verify |
| #176 | [MEDIUM] Potentially disconnected package/module — | `rd-8fec2eee8721a604` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 81 | link_canonical | pending_verify |
| #177 | [MEDIUM] Possible suspicious code island — disconn | `rd-999ad0b7228342bd` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 82 | link_canonical | pending_verify |
| #178 | [MEDIUM] Potential reliability issue: ignored erro | `rd-280d0f0fe459669f` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 83 | link_canonical | pending_verify |
| #179 | [MEDIUM] Potentially disconnected package/module — | `rd-762f59ef98441dd1` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 84 | link_canonical | pending_verify |
| #180 | [MEDIUM] Possible suspicious code island — disconn | `rd-854f7cad983b2393` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #181 | [MEDIUM] Potential reliability issue: ignored erro | `rd-6d38bc4d3577c0d6` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 86 | link_canonical | pending_verify |
| #182 | [MEDIUM] Potentially disconnected package/module — | `rd-38b74bfddbc006fd` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #183 | [MEDIUM] Potentially disconnected package/module — | `rd-a4104968550edb1c` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 89 | link_canonical | pending_verify |
| #184 | [MEDIUM] Potential reliability issue: ignored erro | `rd-80bd0c20a2fa9d1f` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 90 | link_canonical | pending_verify |
| #185 | [MEDIUM] Potential reliability issue: ignored erro | `rd-11c7e16270f942db` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 91 | link_canonical | pending_verify |
| #186 | [MEDIUM] Potential reliability issue: ignored erro | `rd-567499d75f8db670` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 92 | link_canonical | pending_verify |
| #187 | [MEDIUM] Potential reliability issue: ignored erro | `rd-14ab8e7406280f9e` | reliability (AI auditor) | medium | present | duplicate_existing_fingerprint | 120 | link_canonical | unverified |
| #188 | [MEDIUM] Potential reliability issue: ignored erro | `rd-1a163256ff13cf90` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 121 | link_canonical | pending_verify |
| #189 | [MEDIUM] Potential reliability issue: ignored erro | `rd-0170bee0566f8e87` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 122 | link_canonical | pending_verify |
| #190 | [MEDIUM] Potential reliability issue: ignored erro | `rd-f4baf545376d3fe4` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 123 | link_canonical | pending_verify |
| #191 | [MEDIUM] Possible suspicious code island — disconn | `rd-818b04906e147e6c` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #192 | [MEDIUM] Potential reliability issue: ignored erro | `rd-a979086a6e89029f` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 124 | link_canonical | pending_verify |
| #193 | [MEDIUM] Potentially disconnected package/module — | `rd-fe0e03aae1a07b91` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #194 | [MEDIUM] Potentially disconnected package/module — | `rd-021a191dccab877d` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 125 | link_canonical | pending_verify |
| #195 | [MEDIUM] Possible suspicious code island — disconn | `rd-f8816f9138531457` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 127 | link_canonical | pending_verify |
| #196 | [MEDIUM] Potential reliability issue: ignored erro | `rd-d0a05cf6594a828f` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 128 | link_canonical | pending_verify |
| #197 | [MEDIUM] Potentially disconnected package/module — | `rd-e19cc7ae715ca576` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #198 | [MEDIUM] Potential reliability issue: ignored erro | `rd-17ce603a440b2a8a` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 130 | link_canonical | pending_verify |
| #199 | [MEDIUM] Potential reliability issue: ignored erro | `rd-d391ab02bcc6f715` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 131 | link_canonical | pending_verify |
| #200 | [MEDIUM] Potentially disconnected package/module — | `rd-3911ff06c4252097` | graph (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #201 | [MEDIUM] Potentially disconnected package/module — | `rd-d608271f97300c5f` | graph (AI auditor) | medium | absent | duplicate_existing_fingerprint | 98 | link_canonical | pending_verify |
| #202 | Code Review Summary - 79 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #203 | [MEDIUM] Ensure that a user for the container has  | `rd-69bd7b5f1d9d1f8b` | checkov (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #204 | [MEDIUM] Ensure that a user for the container has  | `rd-46267a891b3e317b` | checkov (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #205 | [MEDIUM] Base64 High Entropy String | `rd-a7fb8b9ed08e7f8f` | checkov (AI auditor) | medium | present | active_present_in_latest_scan | 205 | fix_in_code_batch | unverified |
| #206 | [MEDIUM] Base64 High Entropy String | `rd-08c311b79f888ecb` | checkov (AI auditor) | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #207 | [HIGH] Image user should not be 'root' | `rd-9d195d73abbbfea8` | trivy | high | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #208 | [HIGH] Image user should not be 'root' | `rd-7e6fdfc4a33cd692` | trivy | high | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #209 | [MEDIUM] Pin versions in apk add. Instead of `apk  | `rd-3bf46c2bca2f7681` | hadolint (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #210 | [MEDIUM] Pin versions in apk add. Instead of `apk  | `rd-9a5f827df84432c7` | hadolint (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #211 | [MEDIUM] Potential reliability issue: ignored erro | `rd-5db7a89fa16b80ea` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 61 | link_canonical | pending_verify |
| #212 | [MEDIUM] Potential reliability issue: ignored erro | `rd-01184640a65833ca` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 134 | link_canonical | pending_verify |
| #213 | [MEDIUM] Potential reliability issue: ignored erro | `rd-52ca71ddebab9092` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 136 | link_canonical | pending_verify |
| #214 | [MEDIUM] Potential reliability issue: ignored erro | `rd-89bda9c93b10f19b` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 137 | link_canonical | pending_verify |
| #215 | [MEDIUM] Potential reliability issue: ignored erro | `rd-8a66eeb95c8d98f0` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 140 | link_canonical | pending_verify |
| #216 | [MEDIUM] Potential reliability issue: ignored erro | `rd-155765bc2cb8e288` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 141 | link_canonical | pending_verify |
| #217 | [MEDIUM] Potential reliability issue: ignored erro | `rd-4aad6eddd56a9488` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 150 | link_canonical | pending_verify |
| #218 | [MEDIUM] Potential reliability issue: ignored erro | `rd-d4357de4e096411a` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #219 | Code Review Summary - 34 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #220 | Code Review Summary - 28 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #221 | [MEDIUM] Possible internal infrastructure referenc | `rd-9a6eaeee6abc7aa7` | static | medium | present | duplicate_existing_fingerprint | 53 | link_canonical | unverified |
| #222 | [MEDIUM] Possible internal infrastructure referenc | `rd-b67022bb59a95c9b` | static | medium | present | duplicate_existing_fingerprint | 66 | link_canonical | unverified |
| #223 | [MEDIUM] Possible internal infrastructure referenc | `rd-bc82928a9cc02518` | static | medium | present | duplicate_existing_fingerprint | 143 | link_canonical | unverified |
| #224 | [MEDIUM] Possible internal infrastructure referenc | `rd-c073c4b84ab723fb` | static | medium | present | duplicate_existing_fingerprint | 144 | link_canonical | unverified |
| #225 | [MEDIUM] Possible internal infrastructure referenc | `rd-10d09a1086712459` | static | medium | present | duplicate_existing_fingerprint | 145 | link_canonical | unverified |
| #226 | Code Review Summary - 34 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #227 | Code Review Summary - 34 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #228 | [MEDIUM] Base64 High Entropy String | `rd-a42825dca7b2b254` | checkov (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #229 | [LOW] Large commented-out code block | `rd-68e97a67b7fb5e10` | tech_debt (AI auditor) | low | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #230 | [MEDIUM] Pin versions in apk add. Instead of `apk  | `rd-f89d95d860e0f431` | hadolint (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #231 | [MEDIUM] Pin versions in apk add. Instead of `apk  | `rd-e3898643d06d1765` | hadolint (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #232 | [MEDIUM] Pin versions in apk add. Instead of `apk  | `rd-fd4a569728dcf7bc` | hadolint (AI auditor) | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #233 | [MEDIUM] Pin versions in apk add. Instead of `apk  | `rd-e3584a4f91ff4415` | hadolint (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #234 | [MEDIUM] Pin versions in apk add. Instead of `apk  | `rd-2b43e81d5d93f4ce` | hadolint (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #235 | [MEDIUM] Potential reliability issue: ignored erro | `rd-280d0f0fe459669f` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 83 | link_canonical | pending_verify |
| #235 | [MEDIUM] Potential reliability issue: ignored erro | `rd-280d0f0fe459669f` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 83 | link_canonical | pending_verify |
| #236 | [MEDIUM] Potential reliability issue: ignored erro | `rd-6d38bc4d3577c0d6` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 86 | link_canonical | pending_verify |
| #237 | [MEDIUM] Potential reliability issue: ignored erro | `rd-6905bb78b070c1c5` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #238 | [MEDIUM] Potential reliability issue: ignored erro | `rd-97fd762e5505baae` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #239 | [MEDIUM] Potential reliability issue: ignored erro | `rd-52c4e356ddf75410` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #240 | [MEDIUM] Potential reliability issue: ignored erro | `rd-14ab8e7406280f9e` | reliability (AI auditor) | medium | present | duplicate_existing_fingerprint | 120 | link_canonical | unverified |
| #241 | [MEDIUM] Potential reliability issue: ignored erro | `rd-1a163256ff13cf90` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 121 | link_canonical | unverified |
| #242 | [MEDIUM] Potential reliability issue: ignored erro | `rd-0170bee0566f8e87` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 122 | link_canonical | pending_verify |
| #243 | [MEDIUM] Potential reliability issue: ignored erro | `rd-f4baf545376d3fe4` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 123 | link_canonical | pending_verify |
| #244 | [MEDIUM] Potential reliability issue: ignored erro | `rd-a979086a6e89029f` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 124 | link_canonical | pending_verify |
| #245 | [HIGH] Possible SQL injection via string concatena | `rd-1b3559467d23748c` | static | high | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #246 | Code Review Summary - 29 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #247 | [MEDIUM] Potential reliability issue: ignored erro | `rd-dbae99643ce54c9e` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #248 | [MEDIUM] Potential reliability issue: ignored erro | `rd-3aabb9fcdf82c868` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #249 | [MEDIUM] Potential reliability issue: ignored erro | `rd-d0a05cf6594a828f` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 128 | link_canonical | pending_verify |
| #250 | [MEDIUM] Potential reliability issue: ignored erro | `rd-17ce603a440b2a8a` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 130 | link_canonical | pending_verify |
| #251 | [MEDIUM] Potential reliability issue: ignored erro | `rd-d391ab02bcc6f715` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 131 | link_canonical | pending_verify |
| #252 | Code Review Summary - 27 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #253 | [LOW] Large commented-out code block | `rd-77a9184fcc1de7c1` | tech_debt (AI auditor) | low | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #254 | Code Review Summary - 27 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #255 | Code Review Summary - 27 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #256 | Code Review Summary - 17 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #257 | [MEDIUM] Base64 High Entropy String | `rd-8243b8047e42d000` | checkov (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #258 | [LOW] Large commented-out code block | `rd-5f8becca514b40e9` | tech_debt (AI auditor) | low | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #259 | [MEDIUM] Pin versions in apk add. Instead of `apk  | `rd-2e9bfe809e79bcf0` | hadolint (AI auditor) | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #260 | [MEDIUM] Pin versions in apk add. Instead of `apk  | `rd-0795da088fd0a316` | hadolint (AI auditor) | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #261 | [MEDIUM] Pin versions in apk add. Instead of `apk  | `rd-cb25db2e1eddfde9` | hadolint (AI auditor) | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #262 | [MEDIUM] Pin versions in apk add. Instead of `apk  | `rd-ce523a08645d494c` | hadolint (AI auditor) | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #263 | [MEDIUM] Potential reliability issue: ignored erro | `rd-d056a692fd1bac9d` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #264 | [MEDIUM] Potential reliability issue: ignored erro | `rd-2974aefb37ddc89e` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #265 | [MEDIUM] Potential reliability issue: ignored erro | `rd-387c218566982f4c` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #266 | [MEDIUM] Potential reliability issue: ignored erro | `rd-df2dac8f1e732f73` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #267 | [MEDIUM] Potential reliability issue: ignored erro | `rd-e1686003172885fb` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #268 | [MEDIUM] Potential reliability issue: ignored erro | `rd-f2e10a38d86cef25` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #269 | [MEDIUM] Potential reliability issue: ignored erro | `rd-c5c638f6ce1d42b8` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #270 | [MEDIUM] Potential reliability issue: ignored erro | `rd-e316d4a117e143e7` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #271 | [MEDIUM] Potential reliability issue: ignored erro | `rd-2ae4f55b097c1089` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #272 | Code Review Summary - 20 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #275 | [MEDIUM] Base64 High Entropy String | `rd-a7fb8b9ed08e7f8f` | checkov | medium | present | duplicate_existing_fingerprint | 205 | link_canonical | unverified |
| #276 | [MEDIUM] Potential reliability issue: ignored erro | `rd-174988cbdb2011b7` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #277 | Code Review Summary - 14 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #278 | [MEDIUM] Potential reliability issue: ignored erro | `rd-1d71468a43c21301` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #279 | [MEDIUM] Potential reliability issue: ignored erro | `rd-b00bbe25b3689ff8` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #280 | [MEDIUM] Possible internal infrastructure referenc | `rd-794b38b4d1a0a43a` | static | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #281 | Code Review Summary - 15 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #282 | Code Review Summary - 15 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #283 | Code Review Summary - 14 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #284 | Code Review Summary - 14 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #285 | Code Review Summary - 14 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #286 | Code Review Summary - 14 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #287 | Code Review Summary - 15 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #289 | [MEDIUM] Potential reliability issue: ignored erro | `rd-280d0f0fe459669f` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 83 | link_canonical | pending_verify |
| #290 | [MEDIUM] Potential reliability issue: ignored erro | `rd-6d38bc4d3577c0d6` | reliability (AI auditor) | medium | absent | duplicate_existing_fingerprint | 86 | link_canonical | pending_verify |
| #291 | Code Review Summary - 15 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #292 | [MEDIUM] Potential reliability issue: ignored erro | `rd-65cd4062677c9321` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #293 | Code Review Summary - 11 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #294 | Code Review Summary - 11 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #295 | [LOW] Large commented-out code block | `rd-bc6f72145bde322f` | tech_debt (AI auditor) | low | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #296 | [MEDIUM] Possible internal infrastructure referenc | `rd-54c4fba028826514` | static | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #297 | [MEDIUM] Potential reliability issue: ignored erro | `rd-781b1508ee91f62b` | reliability (AI auditor) | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #298 | Code Review Summary - 12 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #299 | Code Review Summary - 12 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #300 | Code Review Summary - 16 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #301 | [MEDIUM] Potential file inclusion via variable | `rd-7ac01e367871e750` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #302 | [MEDIUM] Potential file inclusion via variable | `rd-79390126afc631c8` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #303 | [MEDIUM] Subprocess launched with a potential tain | `rd-0bfd3f45ed94cd4f` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #304 | [MEDIUM] Expect WriteFile permissions to be 0600 o | `rd-574f04844f6b7c09` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #305 | [MEDIUM] Potential file inclusion via variable | `rd-c083aa29ba214d98` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #306 | [MEDIUM] Expect WriteFile permissions to be 0600 o | `rd-3b76c829b26f7da8` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #307 | [MEDIUM] Potential file inclusion via variable | `rd-0b5f428d1bf01004` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #308 | [MEDIUM] Potential file inclusion via variable | `rd-1804c62e3dd3af05` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #309 | [MEDIUM] Expect WriteFile permissions to be 0600 o | `rd-4ab00b3648f8de39` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #310 | [MEDIUM] Potential file inclusion via variable | `rd-99bce8170166fbc0` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #311 | [MEDIUM] Potential file inclusion via variable | `rd-b33665daa3c49eab` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #312 | [MEDIUM] Potential file inclusion via variable | `rd-26db68286d6589ad` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #313 | [MEDIUM] Potential file inclusion via variable | `rd-c0660264888f3f7a` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #314 | [MEDIUM] Expect directory permissions to be 0750 o | `rd-5cd3998b7eac31b5` | gosec | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #315 | [MEDIUM] Expect directory permissions to be 0750 o | `rd-b0097ad6a676180c` | gosec | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #316 | [HIGH] integer overflow conversion uint64 -> int64 | `rd-32ea466677b98678` | gosec | high | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #317 | [MEDIUM] Potential file inclusion via variable | `rd-0c8b4a1aa15b6495` | gosec | medium | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #318 | [MEDIUM] Potential file inclusion via variable | `rd-6068dc4d0ec4c3e9` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #319 | [MEDIUM] Expect directory permissions to be 0750 o | `rd-db7ecda12786c683` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #320 | [MEDIUM] Expect WriteFile permissions to be 0600 o | `rd-671843a970a78be6` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #321 | [MEDIUM] SQL string formatting | `rd-c277e8b2a3ae0431` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #322 | [MEDIUM] Expect directory permissions to be 0750 o | `rd-ed1c73d74547022b` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #323 | [HIGH] Potential hardcoded credentials | `rd-a668d741a770ea04` | gosec | high | absent | resolved_absent_from_latest_scan |  | evidence_closure | pending_verify |
| #324 | [MEDIUM] The used method does not auto-escape HTML | `rd-f0e175991662ee4e` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #325 | Code Review Summary - 41 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #326 | [LOW] Large commented-out code block | `rd-8a7bf238e8da76f2` | tech_debt (AI auditor) | low | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #327 | [HIGH] Possible SQL injection via string concatena | `rd-22ea3ab4d75a571d` | static | high | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #328 | [HIGH] Possible SQL injection via string concatena | `rd-56df68c874191f41` | static | high | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #329 | [MEDIUM] Potential reliability issue: ignored erro | `rd-ae30a4aa7fad0a04` | reliability (AI auditor) | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #330 | [MEDIUM] Expect directory permissions to be 0750 o | `rd-837f90a24801401d` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #331 | [MEDIUM] Expect directory permissions to be 0750 o | `rd-094959790721052d` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #332 | [MEDIUM] Potential file inclusion via variable | `rd-3e397af3dbef8964` | gosec | medium | present | active_present_in_latest_scan |  | fix_in_code_batch | unverified |
| #333 | Code Review Summary - 41 Issues Found | `` |  |  | unknown | out_of_scope_for_current_batch |  | ignore_summary | unverified |
| #334 | [MEDIUM] Potential reliability issue: ignored erro | `rd-b521b37a3eaf1747` | reliability (AI auditor) | medium | unknown | missing_local_mapping_backfilled |  | backfill_mapping | unverified |
| #335 | [MEDIUM] Potential reliability issue: ignored erro | `rd-c87d40514a506c05` | reliability (AI auditor) | medium | unknown | missing_local_mapping_backfilled |  | backfill_mapping | unverified |
| #336 | [MEDIUM] Potential reliability issue: ignored erro | `rd-370385224afc13ec` | reliability (AI auditor) | medium | unknown | missing_local_mapping_backfilled |  | backfill_mapping | unverified |
| #337 | [MEDIUM] Potential reliability issue: ignored erro | `rd-b918d8cd2ba1b13d` | reliability (AI auditor) | medium | unknown | missing_local_mapping_backfilled |  | backfill_mapping | unverified |
