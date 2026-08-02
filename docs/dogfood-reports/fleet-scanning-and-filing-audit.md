# Fleet scanning and filing audit

Generated: 2026-06-18T01:09:43Z
Mode: **dry-run**

## Fleet summary

- Tracked repositories: **40**
- Scan enabled (webhook/manual): **40**
- Nightly schedule on: **0**
- Nightly schedule off (scan enabled): **40**
- Scheduler-eligible: **0**
- Stale scans (&gt;24h, scan enabled): **33**
- Open findings without mapped forge issue: **6289**
- Mapped open forge issues: **22**

> The nightly calibration learner (`rd-deterministic-daily.sh` at 02:17) is **not** a full-fleet scanner.

## Per-repository

| Repo | Scan | Sched | Filing | Last scan | Stale | Unmapped findings | Mapped issues | Skip reason |
|------|------|-------|--------|-----------|-------|-------------------|---------------|-------------|
| commstech/AI_Money_Maker | on | off | on | 2026-06-03T00:41:35.787874486Z | yes | 70 | 0 | schedule_disabled |
| commstech/AMMBER | on | off | on | 2026-06-09T23:59:35.809418802Z | yes | 318 | 0 | schedule_disabled |
| commstech/Alexa_to_Homeassistant | on | off | on | 2026-06-01T14:34:20.821461276Z | yes | 14 | 0 | schedule_disabled |
| commstech/Repository-Detective | on | off | on | 2026-06-17T17:23:25.958180513Z | no | 2457 | 0 | schedule_disabled |
| commstech/Business | on | off | on | 2026-06-17T12:12:09.215223279Z | no | 137 | 0 | schedule_disabled |
| commstech/Dockhand | on | off | on | 2026-06-17T12:12:18.94524582Z | no | 0 | 0 | schedule_disabled |
| commstech/DriveRepair | on | off | on | 2026-06-02T14:31:58.923467517Z | yes | 77 | 0 | schedule_disabled |
| commstech/EOWM | on | off | on | 2026-06-02T14:17:38.167325823Z | yes | 6 | 0 | schedule_disabled |
| commstech/House_Grocery_AI | on | off | on | 2026-06-05T13:13:20.618141608Z | yes | 109 | 0 | schedule_disabled |
| commstech/Infrastructure_as_Code | on | off | on | 2026-06-17T12:12:31.819871174Z | no | 428 | 20 | schedule_disabled |
| commstech/Kometa_Commstech | on | off | on | 2026-06-02T15:08:50.813968541Z | yes | 17 | 0 | schedule_disabled |
| commstech/Luna-Assist | on | off | on | 2026-06-01T13:53:28.283752625Z | yes | 140 | 0 | schedule_disabled |
| commstech/Maintainerr_AI | on | off | on | 2026-06-03T00:41:32.401541906Z | yes | 104 | 0 | schedule_disabled |
| commstech/OpenClaw | on | off | on | 2026-06-08T23:34:03.310471761Z | yes | 32 | 0 | schedule_disabled |
| commstech/OpenClaw-Config | on | off | on | 2026-06-03T01:10:23.380308283Z | yes | 140 | 0 | schedule_disabled |
| commstech/PCAP_Analyser | on | off | on | 2026-06-12T00:49:52.925428895Z | yes | 16 | 0 | schedule_disabled |
| commstech/Transcribarr | on | off | on | 2026-06-17T12:12:51.913916341Z | no | 67 | 0 | schedule_disabled |
| commstech/Wave_Analyser | on | off | on | 2026-06-17T12:12:45.650123658Z | no | 310 | 0 | schedule_disabled |
| commstech/Wifi_Collector | on | off | on | 2026-06-18T00:57:51.730017314Z | no | 235 | 0 | schedule_disabled |
| commstech/ansible_playbooks | on | off | on | 2026-06-11T00:05:04.644785863Z | yes | 83 | 0 | schedule_disabled |
| commstech/commsnet_mon | on | off | on | 2026-06-02T13:51:17.903496917Z | yes | 41 | 0 | schedule_disabled |
| commstech/commsnet_mon_v1 | on | off | on | 2026-06-01T15:16:12.1882296Z | yes | 35 | 0 | schedule_disabled |
| commstech/commsnet_optimizer | on | off | on | 2026-06-07T17:00:30.273885122Z | yes | 8 | 0 | schedule_disabled |
| commstech/eagle | on | off | on | 2026-06-02T14:32:06.636541085Z | yes | 68 | 0 | schedule_disabled |
| commstech/eagle_terminal | on | off | on | 2026-06-02T15:08:49.185758732Z | yes | 5 | 0 | schedule_disabled |
| commstech/lion_track-it | on | off | on | 2026-06-03T01:05:30.698169223Z | yes | 236 | 0 | schedule_disabled |
| commstech/netmapper | on | off | on | 2026-06-07T17:00:30.241710496Z | yes | 399 | 0 | schedule_disabled |
| commstech/netmon | on | off | on | 2026-06-03T00:49:28.081757598Z | yes | 152 | 0 | schedule_disabled |
| commstech/nettech | on | off | on | 2026-06-02T14:31:46.947903513Z | yes | 55 | 0 | schedule_disabled |
| commstech/nextcloud_scripts | on | off | on | 2026-06-07T13:25:48.437151442Z | yes | 0 | 0 | schedule_disabled |
| commstech/openclaw-workflows | on | off | on | 2026-06-01T14:51:45.520209991Z | yes | 0 | 0 | schedule_disabled |
| commstech/openwebui_AGI | on | off | on | 2026-06-01T15:16:17.339612931Z | yes | 31 | 0 | schedule_disabled |
| commstech/optouter | on | off | on | 2026-06-03T00:41:33.765013046Z | yes | 162 | 0 | schedule_disabled |
| commstech/paperless_thunder | on | off | on | 2026-06-02T15:09:11.476945899Z | yes | 15 | 0 | schedule_disabled |
| commstech/plexinator | on | off | on | 2026-06-01T14:08:54.501016565Z | yes | 19 | 0 | schedule_disabled |
| commstech/rd-filing-scratch | on | off | off | 2026-06-12T02:40:30.464443549Z | yes | 5 | 2 | schedule_disabled |
| commstech/talos_cluster | on | off | on | 2026-06-01T14:11:15.270531244Z | yes | 0 | 0 | schedule_disabled |
| commstech/web-scraper-notes | on | off | on | 2026-06-02T14:17:50.453339359Z | yes | 64 | 0 | schedule_disabled |
| commstech/wiki | on | off | on | 2026-06-02T14:18:02.54335476Z | yes | 143 | 0 | schedule_disabled |
| commstech/wiki_obsidian | on | off | on | 2026-06-01T14:13:07.161743837Z | yes | 91 | 0 | schedule_disabled |

