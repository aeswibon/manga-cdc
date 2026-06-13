const UPDATE_CHECK_MS = 60 * 60 * 1000;

export type ServiceWorkerUpdateHandle = {
  applyUpdate: () => void;
  dispose: () => void;
};

function notifyIfWaiting(
  registration: ServiceWorkerRegistration,
  onUpdateWaiting: () => void,
): void {
  if (registration.waiting && navigator.serviceWorker.controller) {
    onUpdateWaiting();
  }
}

export async function watchServiceWorkerUpdates(
  onUpdateWaiting: () => void,
): Promise<ServiceWorkerUpdateHandle | null> {
  if (!('serviceWorker' in navigator)) {
    return null;
  }

  let registration: ServiceWorkerRegistration;
  try {
    registration = await navigator.serviceWorker.register('/sw.js');
  } catch {
    return null;
  }

  notifyIfWaiting(registration, onUpdateWaiting);

  const onUpdateFound = () => {
    const worker = registration.installing;
    if (!worker) return;

    worker.addEventListener('statechange', () => {
      if (worker.state === 'installed' && navigator.serviceWorker.controller) {
        onUpdateWaiting();
      }
    });
  };

  registration.addEventListener('updatefound', onUpdateFound);

  let refreshing = false;
  const onControllerChange = () => {
    if (refreshing) {
      window.location.reload();
    }
  };
  navigator.serviceWorker.addEventListener('controllerchange', onControllerChange);

  const checkForUpdate = () => {
    void registration.update().catch(() => {});
  };

  const updateInterval = window.setInterval(checkForUpdate, UPDATE_CHECK_MS);

  const onVisibilityChange = () => {
    if (document.visibilityState === 'visible') {
      checkForUpdate();
    }
  };
  document.addEventListener('visibilitychange', onVisibilityChange);
  window.setTimeout(checkForUpdate, 10_000);

  return {
    applyUpdate() {
      const waiting = registration.waiting;
      if (waiting) {
        refreshing = true;
        waiting.postMessage({ type: 'SKIP_WAITING' });
        return;
      }
      window.location.reload();
    },
    dispose() {
      registration.removeEventListener('updatefound', onUpdateFound);
      navigator.serviceWorker.removeEventListener('controllerchange', onControllerChange);
      document.removeEventListener('visibilitychange', onVisibilityChange);
      window.clearInterval(updateInterval);
    },
  };
}
