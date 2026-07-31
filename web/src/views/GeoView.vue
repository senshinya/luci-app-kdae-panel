<script setup lang="ts">
import { NText } from 'naive-ui'
import GeoCard from '../components/versions/GeoCard.vue'
</script>

<template>
  <div class="page-stack">
    <div class="page-toolbar">
      <div>
        <h2>Geo 数据</h2>
        <NText depth="3">维护 dae 路由规则使用的 geoip.dat 与 geosite.dat</NText>
      </div>
    </div>
    <GeoCard />
  </div>
</template>
