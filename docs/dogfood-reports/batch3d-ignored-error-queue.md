# Batch 3d — remaining HEALTH-IGNORED-ERROR queue

Scan baseline: `a8bb4cddd72ab80c`

| Issue | File | Ignored call | Fix |
|------:|------|--------------|-----|
| #329 | issues/forge_client.go | AddIssueLabels | Return error when label attach fails |
| #334–#336 | main.go | ShouldBindJSON | Return HTTP 400 on bind error |
| #337 | main_closure.go | OnScanFinish, AddIssueLabels | Log failures |
| #338–#339 | main_remediation.go | SupersedeRemediationPlans, AddLifecycleEvent | Log failures |
| #340–#343 | main_suppressions.go | AddLifecycleEvent, LoadRepository, AnnotateCalibration | Log failures |

Duplicate open issues (#120, #187, #240) for same fingerprint in main_closure.go were closed during duplicate closeout sprint.
