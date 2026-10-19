export const registerServiceWorker = () => {
  // В разработке worker не должен подменять ответы Vite и мешать HMR.
  if (!import.meta.env.PROD || !('serviceWorker' in navigator)) return;

  window.addEventListener('load', () => {
    void navigator.serviceWorker
      .register('/sw.js', { updateViaCache: 'none' })
      .catch(() => {
        console.warn('Не удалось включить работу приложения без сети.');
      });
  });
};
