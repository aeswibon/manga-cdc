import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'

const app = mount(App, {
  target: document.getElementById('app')!,
})

if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js').catch(() => {})
  })
}

window.addEventListener('load', () => {
  void import('@vercel/analytics').then(({ inject }) => inject())
  void import('@vercel/speed-insights').then(({ injectSpeedInsights }) => injectSpeedInsights())
})

export default app
