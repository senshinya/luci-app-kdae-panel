#!/usr/bin/env bash
# kdae-panel 卸载。一键执行：
#
#   bash -c "$(curl -fsSL https://raw.githubusercontent.com/tuoro/kdae-panel/main/scripts/uninstall.sh)"
#   （须在 root shell 中执行）
#
# 默认保留配置、账户数据库与全部配置备份；要连数据一并清除：
#
#   KDAE_PANEL_PURGE=true bash -c "$(curl -fsSL ...)"
#
# 两种模式都不会触碰 dae：它的服务、二进制、配置与 geo 数据全部原样保留。
set -euo pipefail

purge=${KDAE_PANEL_PURGE:-}
if [[ -n ${purge} && ${purge} != true ]]; then
  echo "KDAE_PANEL_PURGE 只接受 true（当前值：'${purge}'）；保留数据请直接不设置该变量" >&2
  exit 1
fi

if [[ ${EUID} -ne 0 ]]; then
  echo "请使用 root 权限运行卸载脚本" >&2
  exit 1
fi

if [[ ! -f /usr/bin/kdae-panel && ! -f /etc/systemd/system/kdae-panel.service ]]; then
  echo "未发现已安装的 kdae-panel（可能已经卸载过），继续清理可能的残留。"
fi

# 数据路径可能被 env 文件改写到默认目录之外，必须在删除 /etc/kdae-panel
# 之前把实际值读出来，否则清理会漏掉被挪走的那些文件。
env_file=/etc/kdae-panel/kdae-panel.env
config_of() {
  [[ -f ${env_file} ]] || return 0
  local value
  value=$(sed -n "s/^$1=//p" "${env_file}" | tail -n1)
  # 与 systemd 的 EnvironmentFile 语义对齐：面板经 systemd 读到的值已被剥掉
  # 成对引号、修剪掉含 \r 在内的首尾空白。这里必须做同样的事，否则
  # KEY="/srv/…" 或在 Windows 上编辑过的 env 会让清理静默漏删。
  value=${value//$'\r'/}
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  if [[ ${#value} -ge 2 ]]; then
    case ${value} in
    \"*\" | \'*\') value=${value:1:-1} ;;
    esac
  fi
  printf '%s' "${value}"
}

state_file=$(config_of KDAE_PANEL_INSTALL_STATE_FILE)
state_file=${state_file:-/var/lib/kdae-panel/dae-install.json}
dae_binary=$(config_of KDAE_PANEL_DAE_BINARY)
dae_binary=${dae_binary:-/usr/bin/dae}
dae_service=$(config_of KDAE_PANEL_SERVICE_NAME)
dae_service=${dae_service:-dae}
# "dae 是否由面板安装"要在数据被清掉之前判断，之后就无从知道了。
panel_installed_dae=""
if [[ -f ${dae_binary} && -f ${state_file} ]]; then
  panel_installed_dae=1
fi

systemctl disable --now kdae-panel.service 2>/dev/null || true
rm -f /etc/systemd/system/kdae-panel.service /usr/bin/kdae-panel
# unit 已不存在时继续保留 drop-in 没有意义，而且自升级的 ReadWritePaths=/usr/bin
# 会在日后重装时悄悄恢复高权限。卸载程序时一并清掉全部 unit override。
rm -rf /etc/systemd/system/kdae-panel.service.d
# 安装时落盘的卸载脚本副本一并清掉：卸载不该留下程序痕迹。
# 删除正在执行的脚本是安全的——rm 只是解除目录项，bash 持有的文件描述符
# 仍指向原 inode；注意 bash 是边读边执行的，原地改写运行中的脚本并不安全，
# 这里只删不写。一键 curl 方式执行时内容来自内存字符串，更与此文件无关。
rm -rf /usr/share/kdae-panel
systemctl daemon-reload 2>/dev/null || true

echo "程序与 systemd 单元已移除。dae 本身不受影响。"

if [[ ${purge} == true ]]; then
  database=$(config_of KDAE_PANEL_DATABASE)
  backup_dir=$(config_of KDAE_PANEL_BACKUP_DIR)
  schedule_file=$(config_of KDAE_PANEL_SCHEDULE_FILE)
  geo_state=$(config_of KDAE_PANEL_GEO_STATE_FILE)
  geo_schedule=$(config_of KDAE_PANEL_GEO_SCHEDULE_FILE)
  geo_sources=$(config_of KDAE_PANEL_GEO_SOURCES_FILE)
  panel_backup=$(config_of KDAE_PANEL_BACKUP_FILE)
  github_token=$(config_of KDAE_PANEL_GITHUB_TOKEN_FILE)
  database=${database:-/var/lib/kdae-panel/panel.db}
  backup_dir=${backup_dir:-/var/lib/kdae-panel/backups}
  schedule_file=${schedule_file:-/var/lib/kdae-panel/schedule.json}
  geo_state=${geo_state:-/var/lib/kdae-panel/geo-update.json}
  geo_schedule=${geo_schedule:-/var/lib/kdae-panel/geo-schedule.json}
  geo_sources=${geo_sources:-/var/lib/kdae-panel/geo-sources.json}
  panel_backup=${panel_backup:-/var/lib/kdae-panel/kdae-panel.previous}
  github_token=${github_token:-/var/lib/kdae-panel/github-token}

  # 备份目录来自 env 配置，又要以 root 递归删除。只有位于默认数据目录之内的
  # 路径才自动删（那里本来就会被整体清掉），其余一律留给用户手工处理——
  # env 里一个手滑的取值不该变成 root 下的 rm -rf。
  backup_dir_skipped=""
  purge_backup_dir() {
    local dir="$1"
    # 归一化：压缩重复斜杠、去掉尾部斜杠，防止 "/var/"、"//var" 之类变体绕过判断
    while [[ ${dir} == *//* ]]; do dir=${dir//\/\//\/}; done
    while [[ ${dir} == */ && ${dir} != / ]]; do dir=${dir%/}; done
    case "${dir}" in
    *"/../"* | */.. | *"/./"* | */. | [!/]* | "")
      echo "备份目录取值可疑（'$1'），已跳过删除，请自行确认后手工处理" >&2
      backup_dir_skipped=$1
      ;;
    /var/lib/kdae-panel | /var/lib/kdae-panel/*)
      rm -rf "${dir}"
      ;;
    *)
      echo "备份目录 '$1' 不在默认数据目录 /var/lib/kdae-panel 内，为防误删已跳过；确认后请手工删除" >&2
      backup_dir_skipped=$1
      ;;
    esac
  }

  # 先删按配置挪到别处的文件，再删两个默认目录；路径在默认目录里时前者是空操作。
  rm -f "${database}" "${schedule_file}" "${geo_state}" "${geo_schedule}" "${geo_sources}" "${panel_backup}" "${github_token}" \
    "${state_file}" "${state_file}.previous" \
    "${state_file}.previous-dae" "${state_file}.previous-dae.pending"
  purge_backup_dir "${backup_dir}"
  rm -rf /etc/kdae-panel /var/lib/kdae-panel

  # 删完逐个核对：将来提取逻辑若再与 systemd 语义出现偏差，
  # 也不能以"删净"的虚假声明收场。
  for leftover in "${database}" "${schedule_file}" "${geo_state}" "${geo_schedule}" "${geo_sources}" \
    "${panel_backup}" "${github_token}" "${state_file}"; do
    if [[ -n ${leftover} && -e ${leftover} ]]; then
      echo "警告：${leftover} 未能删除，请自行确认后手工处理" >&2
    fi
  done

  if [[ -n ${backup_dir_skipped} ]]; then
    echo "已删除配置与账户数据库（/etc/kdae-panel、/var/lib/kdae-panel）。"
    echo "注意：备份目录 '${backup_dir_skipped}' 未删除（取值在默认数据目录之外），请自行确认后手工处理。"
  else
    echo "已一并删除配置、账户数据库与全部配置备份（/etc/kdae-panel、/var/lib/kdae-panel）。"
  fi
  echo "dae 的配置目录 /etc/dae 与 geo 数据未被触碰。"
else
  echo "配置和账户数据未删除；默认位于 /etc/kdae-panel 与 /var/lib/kdae-panel，自定义路径按 env 配置保留。"
  echo "要连数据一并清除，用清除模式重跑（本地脚本副本已随程序移除）："
  echo "  （在 root shell 中）KDAE_PANEL_PURGE=true bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/tuoro/kdae-panel/main/scripts/uninstall.sh)\""
  echo "  （源码检出仍在时等效：sudo KDAE_PANEL_PURGE=true ./scripts/uninstall.sh）"

  for leftover in "${state_file}.previous-dae" "${state_file}.previous-dae.pending"; do
    if [[ -f ${leftover} ]]; then
      echo "其中包含用于回滚的 dae 副本 ${leftover}（$(du -h "${leftover}" | cut -f1)），不再需要时可一并删除。"
    fi
  done
fi

# /etc/dae 下常驻的 geo 数据加起来有几十兆——可能是面板首次安装写入的，
# 也可能是用户或别的工具放的。卸载脚本不动它们（dae 还要用），
# 但要说清位置与大小，否则没人知道这些空间去了哪里。
for geo in /etc/dae/geoip.dat /etc/dae/geosite.dat; do
  if [[ -f ${geo} ]]; then
    echo "/etc/dae 下的 geo 数据 ${geo}（$(du -h "${geo}" | cut -f1)）仍保留，dae 仍在使用它。"
  fi
done

if [[ -n ${panel_installed_dae} ]]; then
  echo "若 dae 本身也是由面板安装的，它仍在 ${dae_binary} 并由 ${dae_service%.service}.service 管理，本脚本不会移除。"
fi
