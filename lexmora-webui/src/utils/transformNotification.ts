const STORAGE_KEY = 'lexmora_transform_finished_notify'

export function isTransformFinishedNotifyEnabled(): boolean {
  if (typeof localStorage === 'undefined') return true
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === null) return true
  return stored === '1'
}

export function setTransformFinishedNotifyEnabled(enabled: boolean): void {
  localStorage.setItem(STORAGE_KEY, enabled ? '1' : '0')
}

export function transformNotificationSupported(): boolean {
  return typeof Notification !== 'undefined'
}

export async function ensureTransformNotificationPermission(): Promise<NotificationPermission | 'unsupported'> {
  if (!transformNotificationSupported()) return 'unsupported'
  if (Notification.permission !== 'default') return Notification.permission
  return Notification.requestPermission()
}

function notificationIconUrl(): string {
  const base = import.meta.env.BASE_URL || '/'
  const normalized = base.endsWith('/') ? base : `${base}/`
  return `${normalized}pwa-192x192.png`
}

export function notifyTransformFinished(operationLabel: string): void {
  if (!isTransformFinishedNotifyEnabled()) return
  if (!transformNotificationSupported() || Notification.permission !== 'granted') return

  const label = operationLabel.trim() || 'Transform'
  try {
    new Notification('Lexmora', {
      body: `${label} finished.`,
      tag: 'lexmora-transform-finished',
      icon: notificationIconUrl(),
    })
  } catch {
    // Browser blocked or Notification ctor failed
  }
}

export async function prepareTransformFinishedNotification(): Promise<void> {
  if (!isTransformFinishedNotifyEnabled()) return
  await ensureTransformNotificationPermission()
}
