typedef unsigned char BYTE;
typedef unsigned short wchar_t;
typedef unsigned short WORD;
typedef unsigned long DWORD;
typedef long LONG;
typedef long long LONGLONG;
typedef unsigned long long ULONGLONG;
typedef void* PVOID;
typedef void* HANDLE;
typedef void* HMODULE;
typedef void* HWND;
typedef int BOOL;
typedef unsigned int UINT;
typedef unsigned long ULONG;
typedef unsigned long long ULONG_PTR;

#define TRUE 1
#define FALSE 0
#define DLL_PROCESS_ATTACH 1
#define DLL_PROCESS_DETACH 0
#define PAGE_EXECUTE_READWRITE 0x40
#define WCA_ACCENT_POLICY 19
#define OBJID_WINDOW ((LONG)0)

#define IMAGE_DIRECTORY_ENTRY_EXPORT 0
#define IMAGE_NUMBEROF_DIRECTORY_ENTRIES 16

typedef struct { WORD Length; WORD MaximumLength; wchar_t *Buffer; } UNICODE_STRING;
typedef struct { PVOID Flink; PVOID Blink; } LIST_ENTRY;
typedef struct { BYTE Reserved1[8]; PVOID Reserved2[3]; LIST_ENTRY InMemoryOrderModuleList; } PEB_LDR_DATA;
typedef struct { BYTE Reserved1[0x18]; PEB_LDR_DATA *Ldr; } PEB;
typedef struct { LIST_ENTRY InLoadOrderLinks; LIST_ENTRY InMemoryOrderLinks; LIST_ENTRY InInitializationOrderLinks; PVOID DllBase; PVOID EntryPoint; ULONG SizeOfImage; UNICODE_STRING FullDllName; UNICODE_STRING BaseDllName; } LDR_DATA_TABLE_ENTRY;

typedef BOOL (*PFN_VirtualProtect)(PVOID, unsigned long long, DWORD, DWORD*);
typedef BOOL (*PFN_FlushInstructionCache)(HANDLE, PVOID, unsigned long long);
typedef unsigned short (*PFN_GetClassNameW)(HWND, wchar_t*, int);
typedef HWND (*PFN_FindWindowW)(const wchar_t*, const wchar_t*);
typedef unsigned int (*PFN_RegisterWindowMessageW)(const wchar_t*);
typedef ULONG_PTR (*PFN_SendMessageTimeoutW)(HWND, unsigned int, ULONG_PTR, LONGLONG, unsigned int, unsigned int, ULONG_PTR*);

typedef struct {
    DWORD Attrib;
    PVOID pvData;
    ULONG cbData;
} WINDOWCOMPOSITIONATTRIBDATA;

typedef BOOL (*PFN_SetWindowCompositionAttribute)(HWND, const WINDOWCOMPOSITIONATTRIBDATA*);

static PFN_VirtualProtect g_VirtualProtect;
static PFN_FlushInstructionCache g_FlushInstructionCache;
static PFN_GetClassNameW g_GetClassNameW;
static PFN_FindWindowW g_FindWindowW;
static PFN_RegisterWindowMessageW g_RegisterWindowMessageW;
static PFN_SendMessageTimeoutW g_SendMessageTimeoutW;
static unsigned int g_RefreshMessage;
static PFN_SetWindowCompositionAttribute g_Target;
static PFN_SetWindowCompositionAttribute g_TrampolineFn;
static BYTE g_Trampoline[96];
static BYTE g_OriginalBytes[32];
static unsigned int g_PatchLength;
static int g_Installed;


#pragma function(memset)
void *memset(void *ptr, int value, unsigned long long num)
{
    BYTE *p = (BYTE*)ptr;
    BYTE v = (BYTE)value;
    unsigned long long i = 0;
    for (; i < num; ++i) p[i] = v;
    return ptr;
}

static void byte_copy(BYTE *dst, const BYTE *src, unsigned long long len)
{
    for (unsigned long long i = 0; i < len; ++i) dst[i] = src[i];
}

static int ascii_eq(const char *a, const char *b)
{
    while (*a && *b) {
        if (*a != *b) return 0;
        ++a; ++b;
    }
    return *a == *b;
}

static int wide_eq_ci_len(const wchar_t *a, unsigned int aLen, const wchar_t *b)
{
    unsigned int i = 0;
    while (b[i]) {
        if (i >= aLen) return 0;
        wchar_t ca = a[i];
        wchar_t cb = b[i];
        if (ca >= L'A' && ca <= L'Z') ca = (wchar_t)(ca + (L'a' - L'A'));
        if (cb >= L'A' && cb <= L'Z') cb = (wchar_t)(cb + (L'a' - L'A'));
        if (ca != cb) return 0;
        ++i;
    }
    return i == aLen;
}

static int wide_eq(const wchar_t *a, const wchar_t *b)
{
    unsigned int i = 0;
    while (b[i]) {
        wchar_t ca = a[i];
        wchar_t cb = b[i];
        if (!ca) return 0;
        if (ca >= L'A' && ca <= L'Z') ca = (wchar_t)(ca + (L'a' - L'A'));
        if (cb >= L'A' && cb <= L'Z') cb = (wchar_t)(cb + (L'a' - L'A'));
        if (ca != cb) return 0;
        ++i;
    }
    return a[i] == 0;
}

static PEB *current_peb(void)
{
    PEB *peb;
    __asm__ volatile ("movq %%gs:0x60, %0" : "=r"(peb));
    return peb;
}

static PVOID module_by_name(const wchar_t *name)
{
    PEB *peb = current_peb();
    if (!peb || !peb->Ldr) return 0;
    BYTE *ldr = (BYTE*)peb->Ldr;
    LIST_ENTRY *head = (LIST_ENTRY*)(ldr + 0x20);
    LIST_ENTRY *node = (LIST_ENTRY*)head->Flink;
    while (node && node != head) {
        BYTE *entry = (BYTE*)node - 0x10;
        PVOID dllBase = *(PVOID*)(entry + 0x30);
        WORD nameLength = *(WORD*)(entry + 0x58);
        wchar_t *nameBuffer = *(wchar_t**)(entry + 0x60);
        if (dllBase && nameBuffer && wide_eq_ci_len(nameBuffer, (unsigned int)(nameLength / 2), name)) return dllBase;
        node = (LIST_ENTRY*)node->Flink;
    }
    return 0;
}

static PVOID export_by_name(PVOID module, const char *name)
{
    if (!module) return 0;
    BYTE *base = (BYTE*)module;
    LONG pe = *(LONG*)(base + 0x3C);
    BYTE *nt = base + pe;
    WORD magic = *(WORD*)(nt + 0x18);
    DWORD exportRva = 0;
    if (magic == 0x20B) exportRva = *(DWORD*)(nt + 0x88);
    else if (magic == 0x10B) exportRva = *(DWORD*)(nt + 0x78);
    if (!exportRva) return 0;
    BYTE *exp = base + exportRva;
    DWORD numberOfNames = *(DWORD*)(exp + 0x18);
    DWORD addrFuncsRva = *(DWORD*)(exp + 0x1C);
    DWORD addrNamesRva = *(DWORD*)(exp + 0x20);
    DWORD addrOrdsRva = *(DWORD*)(exp + 0x24);
    DWORD *names = (DWORD*)(base + addrNamesRva);
    WORD *ords = (WORD*)(base + addrOrdsRva);
    DWORD *funcs = (DWORD*)(base + addrFuncsRva);
    for (DWORD i = 0; i < numberOfNames; ++i) {
        const char *candidate = (const char*)(base + names[i]);
        if (ascii_eq(candidate, name)) {
            return base + funcs[ords[i]];
        }
    }
    return 0;
}

static int has_modrm(BYTE op)
{
    if (op <= 0x03 || (op >= 0x08 && op <= 0x0B) || (op >= 0x10 && op <= 0x13) || (op >= 0x18 && op <= 0x1B)) return 1;
    if ((op >= 0x20 && op <= 0x23) || (op >= 0x28 && op <= 0x2B) || (op >= 0x30 && op <= 0x33) || (op >= 0x38 && op <= 0x3B)) return 1;
    if (op == 0x62 || op == 0x63 || op == 0x69 || op == 0x6B) return 1;
    if (op >= 0x80 && op <= 0x8F) return 1;
    if (op == 0xC0 || op == 0xC1 || op == 0xC4 || op == 0xC5 || op == 0xC6 || op == 0xC7) return 1;
    if (op >= 0xD0 && op <= 0xD3) return 1;
    if (op >= 0xD8 && op <= 0xDF) return 1;
    if (op == 0xF6 || op == 0xF7 || op == 0xFE || op == 0xFF) return 1;
    return 0;
}

static unsigned int modrm_len(const BYTE *p, int addr32)
{
    BYTE modrm = p[0];
    unsigned int len = 1;
    unsigned int mod = modrm >> 6;
    unsigned int rm = modrm & 7;
    if (addr32 && mod == 0 && rm == 4) len += 1;
    else if (!addr32 && rm == 4 && mod != 3) len += 1;
    if (mod == 0) {
        if (rm == 5) len += 4;
    } else if (mod == 1) {
        len += 1;
    } else if (mod == 2) {
        len += 4;
    }
    return len;
}

static unsigned int instruction_length(const BYTE *p)
{
    unsigned int i = 0;
    int rexw = 0;
    int addr32 = 0;
    for (;;) {
        BYTE b = p[i];
        if (b == 0xF0 || b == 0xF2 || b == 0xF3 || b == 0x2E || b == 0x36 || b == 0x3E || b == 0x26 || b == 0x64 || b == 0x65 || b == 0x66) { ++i; continue; }
        if (b == 0x67) { addr32 = 1; ++i; continue; }
        if (b >= 0x40 && b <= 0x4F) { rexw = (b & 8) != 0; ++i; continue; }
        break;
    }
    BYTE op = p[i++];
    if (op == 0x0F) {
        BYTE op2 = p[i++];
        if (op2 >= 0x80 && op2 <= 0x8F) return i + 4;
        if (op2 == 0x05 || op2 == 0x07 || op2 == 0x0B || op2 == 0x30 || op2 == 0x31 || op2 == 0x32 || op2 == 0x33 || op2 == 0x34 || op2 == 0x35 || op2 == 0x37 || op2 == 0x77 || op2 == 0xA0 || op2 == 0xA1 || op2 == 0xA2 || op2 == 0xA8 || op2 == 0xA9 || op2 == 0xAA) return i;
        if (op2 == 0x38 || op2 == 0x3A) {
            ++i;
            return i + modrm_len(p + i, addr32) + (op2 == 0x3A ? 1 : 0);
        }
        return i + modrm_len(p + i, addr32);
    }
    if (op >= 0x70 && op <= 0x7F) return i + 1;
    if (op == 0xE8 || op == 0xE9) return i + 4;
    if (op == 0xEB) return i + 1;
    if (op == 0x68) return i + 4;
    if (op == 0x6A) return i + 1;
    if (op >= 0xB0 && op <= 0xB7) return i + 1;
    if (op >= 0xB8 && op <= 0xBF) return i + (rexw ? 8 : 4);
    if (op == 0xC2 || op == 0xCA) return i + 2;
    if (op == 0xC8) return i + 3;
    if (op == 0xC9 || op == 0xCB || op == 0xCC || op == 0xCF || op == 0xC3) return i;
    if (op == 0xCD) return i + 1;
    if (op >= 0x50 && op <= 0x5F) return i;
    if (op == 0xA0 || op == 0xA1 || op == 0xA2 || op == 0xA3) return i + (addr32 ? 4 : 8);
    if (op == 0x80 || op == 0x82 || op == 0x83) return i + modrm_len(p + i, addr32) + 1;
    if (op == 0x81) return i + modrm_len(p + i, addr32) + 4;
    if (op == 0x69) return i + modrm_len(p + i, addr32) + 4;
    if (op == 0x6B) return i + modrm_len(p + i, addr32) + 1;
    if (has_modrm(op)) {
        unsigned int n = i + modrm_len(p + i, addr32);
        if (op == 0xC6) n += 1;
        if (op == 0xC7) n += 4;
        if (op == 0xF6) n += 1;
        if (op == 0xF7) n += 4;
        return n;
    }
    return i;
}

static void relocate_instruction(BYTE *dst, const BYTE *src, unsigned int len)
{
    byte_copy(dst, src, len);
    unsigned int i = 0;
    int addr32 = 0;
    for (;;) {
        BYTE b = src[i];
        if (b == 0xF0 || b == 0xF2 || b == 0xF3 || b == 0x2E || b == 0x36 || b == 0x3E || b == 0x26 || b == 0x64 || b == 0x65 || b == 0x66) { ++i; continue; }
        if (b == 0x67) { addr32 = 1; ++i; continue; }
        if (b >= 0x40 && b <= 0x4F) { ++i; continue; }
        break;
    }
    BYTE op = src[i++];
    if (op == 0xE8 || op == 0xE9) {
        int oldDisp = *(const int*)(src + i);
        ULONGLONG target = (ULONGLONG)(src + len) + oldDisp;
        int newDisp = (int)(target - (ULONGLONG)(dst + len));
        *(int*)(dst + i) = newDisp;
        return;
    }
    if (op == 0xEB || (op >= 0x70 && op <= 0x7F)) {
        int oldDisp = *(const signed char*)(src + i);
        ULONGLONG target = (ULONGLONG)(src + len) + oldDisp;
        long long diff = (long long)(target - (ULONGLONG)(dst + len));
        if (diff >= -128 && diff <= 127) dst[i] = (BYTE)(signed char)diff;
        else dst[0] = 0xCC;
        return;
    }
    if (op == 0x0F && src[i] >= 0x80 && src[i] <= 0x8F) {
        ++i;
        int oldDisp = *(const int*)(src + i);
        ULONGLONG target = (ULONGLONG)(src + len) + oldDisp;
        int newDisp = (int)(target - (ULONGLONG)(dst + len));
        *(int*)(dst + i) = newDisp;
        return;
    }
    if (addr32) return;
    if (op == 0x8B || op == 0x89 || op == 0x8D || op == 0x8A || op == 0x88 || op == 0x03 || op == 0x01 || op == 0x2B || op == 0x29 || op == 0x33 || op == 0x31 || op == 0x8F || op == 0xFF || op == 0xF7 || op == 0xC7 || op == 0xC6) {
        BYTE modrm = src[i];
        unsigned int mod = modrm >> 6;
        unsigned int rm = modrm & 7;
        unsigned int modrmBytes = modrm_len(src + i, addr32);
        if (mod == 0 && rm == 5) {
            unsigned int dispOff = i + 1;
            int oldDisp = *(const int*)(src + dispOff);
            ULONGLONG target = (ULONGLONG)(src + len) + oldDisp;
            int newDisp = (int)(target - (ULONGLONG)(dst + len));
            *(int*)(dst + dispOff) = newDisp;
        } else if (mod != 3 && rm == 4) {
            BYTE sib = src[i + 1];
            if ((sib & 7) == 5 && mod == 0) {
                unsigned int dispOff = i + 2;
                int oldDisp = *(const int*)(src + dispOff);
                ULONGLONG target = (ULONGLONG)(src + len) + oldDisp;
                int newDisp = (int)(target - (ULONGLONG)(dst + len));
                *(int*)(dst + dispOff) = newDisp;
            }
        }
        (void)modrmBytes;
    }
}

static int is_taskbar(HWND hwnd)
{
    if (!g_GetClassNameW || !hwnd) return 0;
    wchar_t name[64];
    unsigned short n = g_GetClassNameW(hwnd, name, 64);
    if (!n || n >= 64) return 0;
    name[n] = 0;
    return wide_eq(name, L"Shell_TrayWnd") || wide_eq(name, L"Shell_SecondaryTrayWnd");
}

static BOOL hook_SetWindowCompositionAttribute(HWND hwnd, const WINDOWCOMPOSITIONATTRIBDATA *data)
{
    if (data && data->Attrib == WCA_ACCENT_POLICY && is_taskbar(hwnd) && g_FindWindowW && g_SendMessageTimeoutW && g_RefreshMessage) {
        static const wchar_t windowClass[] = L"PlutonLowBarHiddenWindow";
        HWND lowbar = g_FindWindowW(windowClass, 0);
        if (lowbar) {
            unsigned long long result = 0;
            unsigned long long sendResult = g_SendMessageTimeoutW(lowbar, g_RefreshMessage, 0, 0, 0x0002 | 0x0001 | 0x00000020, 250, &result);
            if (sendResult && result) return TRUE;
        }
    }
    if (!g_TrampolineFn) return FALSE;
    return g_TrampolineFn(hwnd, data);
}

static void write_abs_jump(BYTE *at, const void *target)
{
    at[0] = 0xFF; at[1] = 0x25; at[2] = 0; at[3] = 0; at[4] = 0; at[5] = 0;
    ULONGLONG addr = (ULONGLONG)target;
    byte_copy(at + 6, (const BYTE*)&addr, 8);
}

static int install_hook(void)
{
    if (g_Installed) return 1;
    PVOID kernel32 = module_by_name(L"kernel32.dll");
    PVOID user32 = module_by_name(L"user32.dll");
    if (!kernel32 || !user32) return 0;
    g_VirtualProtect = (PFN_VirtualProtect)export_by_name(kernel32, "VirtualProtect");
    g_FlushInstructionCache = (PFN_FlushInstructionCache)export_by_name(kernel32, "FlushInstructionCache");
    g_GetClassNameW = (PFN_GetClassNameW)export_by_name(user32, "GetClassNameW");
    g_FindWindowW = (PFN_FindWindowW)export_by_name(user32, "FindWindowW");
    g_RegisterWindowMessageW = (PFN_RegisterWindowMessageW)export_by_name(user32, "RegisterWindowMessageW");
    g_SendMessageTimeoutW = (PFN_SendMessageTimeoutW)export_by_name(user32, "SendMessageTimeoutW");
    if (!g_VirtualProtect || !g_FlushInstructionCache || !g_GetClassNameW || !g_FindWindowW || !g_RegisterWindowMessageW || !g_SendMessageTimeoutW) return 0;
    g_Target = (PFN_SetWindowCompositionAttribute)export_by_name(user32, "SetWindowCompositionAttribute");
    if (!g_Target) return 0;
    g_RefreshMessage = g_RegisterWindowMessageW(L"LowBar.ShouldBlockTaskbarComposition");
    if (!g_RefreshMessage) return 0;

    BYTE *target = (BYTE*)g_Target;
    unsigned int n = 0;
    while (n < 14) {
        unsigned int len = instruction_length(target + n);
        if (len == 0 || len > 15 || n + len > sizeof(g_OriginalBytes)) return 0;
        n += len;
    }
    g_PatchLength = n;
    for (unsigned int i = 0; i < 96; ++i) g_Trampoline[i] = 0xCC;
    unsigned int cursor = 0;
    while (cursor < g_PatchLength) {
        unsigned int len = instruction_length(target + cursor);
        relocate_instruction(g_Trampoline + cursor, target + cursor, len);
        byte_copy(g_OriginalBytes + cursor, target + cursor, len);
        cursor += len;
    }
    write_abs_jump(g_Trampoline + g_PatchLength, target + g_PatchLength);
    DWORD trampolineOldProtect = 0;
    if (g_VirtualProtect(g_Trampoline, 96, PAGE_EXECUTE_READWRITE, &trampolineOldProtect) == FALSE) return 0;
    g_TrampolineFn = (PFN_SetWindowCompositionAttribute)(void*)g_Trampoline;

    DWORD oldProtect = 0;
    if (!g_VirtualProtect(target, g_PatchLength, PAGE_EXECUTE_READWRITE, &oldProtect)) return 0;
    write_abs_jump(target, (const void*)&hook_SetWindowCompositionAttribute);
    for (unsigned int i = 14; i < g_PatchLength; ++i) target[i] = 0x90;
    g_FlushInstructionCache((HANDLE)(ULONG_PTR)-1, target, g_PatchLength);
    DWORD ignored = 0;
    g_VirtualProtect(target, g_PatchLength, oldProtect, &ignored);
    g_Installed = 1;
    return 1;
}

static void uninstall_hook(void)
{
    if (!g_Installed || !g_Target) return;
    BYTE *target = (BYTE*)g_Target;
    DWORD oldProtect = 0;
    if (g_VirtualProtect && g_VirtualProtect(target, g_PatchLength, PAGE_EXECUTE_READWRITE, &oldProtect)) {
        byte_copy(target, g_OriginalBytes, g_PatchLength);
        if (g_FlushInstructionCache) g_FlushInstructionCache((HANDLE)(ULONG_PTR)-1, target, g_PatchLength);
        DWORD ignored = 0;
        g_VirtualProtect(target, g_PatchLength, oldProtect, &ignored);
    }
    g_Installed = 0;
}

__declspec(dllexport) DWORD LowBarExplorerHookStop(HMODULE module)
{
    g_Installed = 0;
    uninstall_hook();

    if (module) {
        PVOID kernel32 = module_by_name(L"kernel32.dll");
        PVOID kernelbase = module_by_name(L"kernelbase.dll");
        typedef void (*PFN_FreeLibraryAndExitThread)(HMODULE, DWORD);
        PFN_FreeLibraryAndExitThread freeAndExit = 0;
        if (kernel32) freeAndExit = (PFN_FreeLibraryAndExitThread)export_by_name(kernel32, "FreeLibraryAndExitThread");
        if (!freeAndExit && kernelbase) freeAndExit = (PFN_FreeLibraryAndExitThread)export_by_name(kernelbase, "FreeLibraryAndExitThread");
        if (freeAndExit) freeAndExit(module, 0);
    }
    return 0;
}

BOOL DllMain(PVOID hinst, DWORD reason, PVOID reserved)
{
    (void)hinst; (void)reserved;
    if (reason == DLL_PROCESS_ATTACH) install_hook();
    else if (reason == DLL_PROCESS_DETACH) uninstall_hook();
    return TRUE;
}
