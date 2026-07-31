import { onBeforeUnmount, onMounted, readonly, ref, type Ref } from 'vue'

const mobileMedia = '(max-width: 767px)'

/** 与 style.css 使用同一个断点，只挂载当前设备真正需要的视图。 */
export function useMobileViewport(): Readonly<Ref<boolean>> {
  const query = window.matchMedia(mobileMedia)
  const mobile = ref(query.matches)
  const update = (event: MediaQueryListEvent) => { mobile.value = event.matches }

  onMounted(() => query.addEventListener('change', update))
  onBeforeUnmount(() => query.removeEventListener('change', update))
  return readonly(mobile)
}
