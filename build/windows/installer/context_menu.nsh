; Explorer context menu entries ("PDF Previewで開く"). Included from project.nsi.
;
; The app registers the entries per user (HKCU\Software\Classes) and turns
; them on and off from its "設定" menu (see contextmenu.go). The installer only
; removes them:
; - on install, the all-users entries (HKLM) written by earlier installers,
;   which would otherwise stay even when turned off in the app
; - on uninstall, both the all-users and the current user's entries

!define CONTEXT_MENU_VERB "OpenWithPdfPreview"

; Deletes the entry of one file class under ROOT\Software\Classes
!macro contextMenu.deleteVerb ROOT CLASS
    DeleteRegKey ${ROOT} "Software\Classes\${CLASS}\shell\${CONTEXT_MENU_VERB}"
!macroend

; Deletes the entries of all file classes under ROOT (HKLM or HKCU)
!macro contextMenu.deleteAll ROOT
    !insertmacro contextMenu.deleteVerb ${ROOT} "Directory"
    !insertmacro contextMenu.deleteVerb ${ROOT} "Directory\Background"
    !insertmacro contextMenu.deleteVerb ${ROOT} "SystemFileAssociations\.xlsx"
    !insertmacro contextMenu.deleteVerb ${ROOT} "SystemFileAssociations\.xlsm"
    !insertmacro contextMenu.deleteVerb ${ROOT} "SystemFileAssociations\.xls"
    !insertmacro contextMenu.deleteVerb ${ROOT} "SystemFileAssociations\.docx"
    !insertmacro contextMenu.deleteVerb ${ROOT} "SystemFileAssociations\.doc"
    !insertmacro contextMenu.deleteVerb ${ROOT} "SystemFileAssociations\.pdf"
!macroend

; Tells Explorer that file associations changed, so the menu is updated
!macro contextMenu.notifyShell
    System::Call 'shell32::SHChangeNotify(i 0x08000000, i 0, p 0, p 0)'
!macroend

; On install: remove all-users entries of earlier versions
!macro contextMenu.removeLegacy
    !insertmacro contextMenu.deleteAll HKLM
    !insertmacro contextMenu.notifyShell
!macroend

; On uninstall: remove all entries
!macro contextMenu.removeAll
    !insertmacro contextMenu.deleteAll HKLM
    !insertmacro contextMenu.deleteAll HKCU
    !insertmacro contextMenu.notifyShell
!macroend
