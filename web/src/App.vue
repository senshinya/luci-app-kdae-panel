<script setup lang="ts">
import {
  NConfigProvider,
  NDialogProvider,
  NGlobalStyle,
  NMessageProvider,
  NNotificationProvider,
  darkTheme,
  zhCN,
  dateZhCN,
  type GlobalThemeOverrides,
} from 'naive-ui'
import { RouterView } from 'vue-router'

// 主题色从 style.css 的 :root 令牌读取，两边永远是同一份定义。
// 不写死字面值：否则品牌青与组件库主色（naive-ui 默认薄荷绿）会各说各话，
// 页面上同时出现两种强调色。
const rootTokens = getComputedStyle(document.documentElement)
const token = (name: string) => rootTokens.getPropertyValue(name).trim()

const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: token('--accent'),
    primaryColorHover: token('--accent-hover'),
    primaryColorPressed: token('--accent-pressed'),
    primaryColorSuppl: token('--accent'),
    bodyColor: token('--bg-page'),
    cardColor: token('--bg-raised'),
    // Select 等浮动菜单会传送到页面根节点，不能依赖所在卡片的背景。
    // 明确指定弹层表面，避免菜单只剩文字、看起来像透明背景。
    popoverColor: token('--bg-raised'),
    fontFamily: token('--font-sans'),
    fontFamilyMono: token('--font-mono'),
    // 自绘元素（outline 卡片、brand-mark）用的是 6-8px 圆角，
    // 组件库默认 3px 与之并存会显得两套体系，统一到 6px
    borderRadius: '6px',
  },
}
</script>

<template>
  <NConfigProvider :theme="darkTheme" :theme-overrides="themeOverrides" :locale="zhCN" :date-locale="dateZhCN">
    <NGlobalStyle />
    <NMessageProvider>
      <NDialogProvider>
        <NNotificationProvider>
          <RouterView />
        </NNotificationProvider>
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>
