import { onMounted, ref, watch } from 'vue'

export function useRememberRef<T>(key: string, value: T) {
  const r = ref<T>(localStorage[key] ? JSON.parse(localStorage[key]) : value)
  watch(r, (v) => {
    localStorage[key] = JSON.stringify(v)
  })
  return r
}
