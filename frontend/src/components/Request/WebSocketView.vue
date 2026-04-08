<template>
  <div style="height: 100%; position: relative; display: flex; flex-direction: column;">
    <!-- 顶部工具栏 -->
    <div class="ws-toolbar" v-if="chatMode">
      <div class="ws-filter-btns">
        <button :class="['ws-btn', dirFilter === 'all' ? 'ws-btn-active' : '']" @click="dirFilter='all'">全部</button>
        <button :class="['ws-btn', dirFilter === 'send' ? 'ws-btn-send' : '']" @click="dirFilter='send'">↑ 发送</button>
        <button :class="['ws-btn', dirFilter === 'recv' ? 'ws-btn-recv' : '']" @click="dirFilter='recv'">↓ 接收</button>
      </div>
      <input class="ws-search" v-model="searchKey" placeholder="搜索消息内容..." />
      <button class="ws-btn ws-btn-toggle" @click="chatMode=false" title="切换到列表视图">☰</button>
    </div>
    <!-- 统计栏 -->
    <div class="ws-stats" v-if="chatMode">
      <span>总计: {{ statsTotal }} 条</span>
      <span>发送: {{ statsSend }} ({{ formatBytes(statsSendBytes) }})</span>
      <span>接收: {{ statsRecv }} ({{ formatBytes(statsRecvBytes) }})</span>
    </div>
    <!-- 聊天视图 (虚拟滚动) -->
    <div class="ws-chat-container" v-if="chatMode" ref="chatContainer" @scroll="onChatScroll">
      <div v-if="filteredMessages.length === 0" class="ws-empty">暂无发送、接收的数据</div>
      <div v-else :style="{height: filteredMessages.length * itemHeight + 'px', position: 'relative'}">
        <div v-for="(msg, vi) in visibleMessages" :key="msg._key"
             :style="{position: 'absolute', top: msg._top + 'px', left: 0, right: 0}"
             :class="['ws-bubble-row', isSend(msg) ? 'ws-send' : 'ws-recv', selectedIdx === msg._idx ? 'ws-selected' : '']"
             @click="selectMessage(msg._idx, msg)">
          <div class="ws-bubble">
            <div class="ws-bubble-header">
              <span class="ws-dir-tag">{{ isSend(msg) ? 'SEND' : 'RECV' }}</span>
              <span class="ws-type-tag" v-if="msg['类型']">[{{ msg['类型'] }}]</span>
              <span class="ws-len">{{ msg['长度'] }} 字节</span>
              <span class="ws-time">{{ msg['时间'] }}</span>
            </div>
            <div class="ws-bubble-body">{{ truncate(msg['数据'], 200) }}</div>
          </div>
        </div>
      </div>
    </div>
    <!-- 列表模式切换按钮 -->
    <div v-if="!chatMode" style="position:absolute;top:2px;right:4px;z-index:10;">
      <button class="ws-btn ws-btn-toggle" @click="chatMode=true" title="切换到聊天视图">💬</button>
    </div>
    <!-- 原 ag-Grid 视图 -->
    <ag-grid-vue
        v-show="!chatMode"
        ref="agGrid"
        style="height: 100%; flex: 1;"
        :defaultColDef="defaultColDef"
        :rowData="RowData"
        :columnDefs="columns"
        :enableRangeSelection="true"
        :enableCharts="true"
        :modules="leftModules"
        :grid-options="gridOptions"
        :overlayNoRowsTemplate="overlayNoRowsTemplate"
    >
    </ag-grid-vue>
    <div v-show="LineSelected"/>
  </div>
</template>

<script>
import '@ag-grid-community/styles/ag-grid.css';
import '@ag-grid-community/styles/ag-theme-balham.css';
import {AgGridVue} from '@ag-grid-community/vue3';
import {ClipboardModule} from '@ag-grid-enterprise/clipboard';
import {SetFilterModule} from '@ag-grid-enterprise/set-filter';
import {ExcelExportModule} from '@ag-grid-enterprise/excel-export';
import ImageRenderer from './SocketImage.vue';
import {CallGoDo} from "../CallbackEventsOn.js";

function isSendIco(ico) {
  return ico === '上行' || ico === '拦截上行'
}

export default {
  props: ['readOnly', 'width'],
  watch: {
    readOnly(value) {
      this.ReadOnly = value
    },
    width(value) {
      this._Width = parseInt(value.replaceAll("px", ""))
    },
  },
  components: {
    'ag-grid-vue': AgGridVue, imageRenderer: ImageRenderer,
  },
  computed: {
    filteredMessages() {
      let msgs = this.allMessages
      if (this.dirFilter === 'send') {
        msgs = msgs.filter(m => isSendIco(m.ico))
      } else if (this.dirFilter === 'recv') {
        msgs = msgs.filter(m => !isSendIco(m.ico))
      }
      if (this.searchKey) {
        const key = this.searchKey.toLowerCase()
        msgs = msgs.filter(m => (m['数据'] || '').toLowerCase().includes(key))
      }
      return msgs
    },
    visibleMessages() {
      const msgs = this.filteredMessages
      const buffer = 5
      const startIdx = Math.max(0, Math.floor(this.scrollTop / this.itemHeight) - buffer)
      const visibleCount = Math.ceil(this.containerHeight / this.itemHeight) + buffer * 2
      const endIdx = Math.min(msgs.length, startIdx + visibleCount)
      const result = []
      for (let i = startIdx; i < endIdx; i++) {
        const m = msgs[i]
        result.push({...m, _idx: i, _top: i * this.itemHeight, _key: m['#'] + '-' + i})
      }
      return result
    },
    statsTotal() { return this.allMessages.length },
    statsSend() { return this.allMessages.filter(m => isSendIco(m.ico)).length },
    statsRecv() { return this.allMessages.filter(m => !isSendIco(m.ico)).length },
    statsSendBytes() {
      return this.allMessages.filter(m => isSendIco(m.ico)).reduce((s, m) => s + (parseInt(m['长度']) || 0), 0)
    },
    statsRecvBytes() {
      return this.allMessages.filter(m => !isSendIco(m.ico)).reduce((s, m) => s + (parseInt(m['长度']) || 0), 0)
    },
    LineSelected() {
      this.MenuItems[6].subMenu[0].disabled = this.agSelectedLine === null
      if (this.darkTheme) {
        for (let i = 0; i < this.MenuItems.length; i++) {
          if (typeof this.MenuItems[i] === 'string') {
            continue
          }
          if (this.MenuItems[i].selected) {
            this.MenuItems[i].icon = `<svg xmlns="http://www.w3.org/2000/svg" style="top: 2px;position: relative;" width="16" height="14" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">\\n' +
              '    <polyline points="20 6 9 17 4 12"/>' +
              '</svg>`
          } else {
            this.MenuItems[i].icon = ""
          }
        }
      } else {
        for (let i = 0; i < this.MenuItems.length; i++) {
          if (typeof this.MenuItems[i] === 'string') {
            continue
          }
          if (this.MenuItems[i].selected) {
            this.MenuItems[i].icon = `<svg xmlns="http://www.w3.org/2000/svg" style="top: 2px;position: relative;" width="16" height="14" viewBox="0 0 24 24" fill="none" stroke="#000" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">\\n' +
              '    <polyline points="20 6 9 17 4 12"/>' +
              '</svg>`
          } else {
            this.MenuItems[i].icon = ""
          }
        }
      }
      window.Socket.Line = this.agSelectedLine
      return false
    },
  },
  data() {
    return {
      chatMode: true,
      dirFilter: 'all',
      searchKey: '',
      allMessages: [],
      selectedIdx: -1,
      scrollTop: 0,
      containerHeight: 400,
      itemHeight: 60,
      _Width: 0,
      IsHasModify: false,
      agSelectedLine: null,
      overlayNoRowsTemplate: `<span style="padding: 20px;" id="HookMessageText">暂无发送、接收的数据</span>`,
      leftModules: [SetFilterModule, ClipboardModule],
      rightModules: [ExcelExportModule],
      gridOptions: {
        onRangeSelectionChanged: this.onRangeSelectionChanged,
        onRowClicked: this.onRowClicked,
        onCellFocused: this.onCellFocused,
        getContextMenuItems: this.onContextMenuItems,
        getRowStyle: this.onGetRowStyle,
        onRowDataUpdated: this.NewColumnsLoaded,
        onModelUpdated: this.NewColumnsLoaded,
        onCellValueChanged: (event) => {
          this.IsHasModify = true
        },
        suppressScrollOnNewData: true,
      },
      defaultColDef: {
        flex: 1,
        minWidth: 10,
        sortable: false,
        filter: true,
        floatingFilter: false,
        resizable: true,
        menuTabs: ['filterMenuTab'],
        suppressNavigable: false,
        cellClass: 'no-border'
      },
      MenuItems: [
        {
          name: '跟随显示',
          action: () => {
            this.MenuItems[0].selected = !this.MenuItems[0].selected
          },
          disabled: false,
          selected: true
        },
        'separator',
        {
          name: '停止插入发送',
          action: () => {
            this.MenuItems[2].selected = !this.MenuItems[2].selected
            if (this.MenuItems[2].selected) {
              this.MenuItems[4].selected = false
            }
            let way = window.vm.List.agSelectedLine.data["方式"].toUpperCase()
            CallGoDo("设置右键菜单配置", {
              StopSend: this.MenuItems[2].selected,
              StopRec: this.MenuItems[3].selected,
              StopALL: this.MenuItems[4].selected,
              Theology: window.Theology,
              IsWs: window.vm.Tabs.Request.DisplayHTTPHeader,
              IsTCP: way.indexOf("TCP") !== -1
            })
          },
          disabled: false,
          selected: false
        },
        {
          name: '停止插入接收',
          action: () => {
            this.MenuItems[3].selected = !this.MenuItems[3].selected
            if (this.MenuItems[3].selected) {
              this.MenuItems[4].selected = false
            }
            let way = window.vm.List.agSelectedLine.data["方式"].toUpperCase()
            CallGoDo("设置右键菜单配置", {
              StopSend: this.MenuItems[2].selected,
              StopRec: this.MenuItems[3].selected,
              StopALL: this.MenuItems[4].selected,
              Theology: window.Theology,
              IsWs: window.vm.Tabs.Request.DisplayHTTPHeader,
              IsTCP: way.indexOf("TCP") !== -1
            })
          },
          disabled: false,
          selected: false
        },
        {
          name: '全部停止插入',
          action: () => {
            this.MenuItems[4].selected = !this.MenuItems[4].selected
            if (this.MenuItems[4].selected) {
              this.MenuItems[2].selected = false
              this.MenuItems[3].selected = false
            }
            let way = window.vm.List.agSelectedLine.data["方式"].toUpperCase()
            CallGoDo("设置右键菜单配置", {
              StopSend: this.MenuItems[2].selected,
              StopRec: this.MenuItems[3].selected,
              StopALL: this.MenuItems[4].selected,
              Theology: window.Theology,
              IsWs: window.vm.Tabs.Request.DisplayHTTPHeader,
              IsTCP: way.indexOf("TCP") !== -1
            })
          },
          disabled: false,
          selected: false
        },
        'separator',
        {
          name: '复制',
          subMenu: [
            {
              name: '复制选中HEX到剪辑版',
              action: () => {
                let way = window.vm.List.agSelectedLine.data["方式"].toUpperCase()
                CallGoDo("socket点击右键菜单", {
                  Type: "Selected",
                  Theology: window.Theology,
                  IsWs: window.vm.Tabs.Request.DisplayHTTPHeader,
                  IsTCP: way.indexOf("TCP") !== -1,
                  SelectedID: this.agSelectedLine.data["#"]
                })
              },
              disabled: false,
              visible: true
            },
            {
              name: '复制所有HEX到剪辑版',
              action: () => {
                let way = window.vm.List.agSelectedLine.data["方式"].toUpperCase()
                CallGoDo("socket点击右键菜单", {
                  Type: "AllHEX",
                  Theology: window.Theology,
                  IsWs: window.vm.Tabs.Request.DisplayHTTPHeader,
                  IsTCP: way.indexOf("TCP") !== -1,
                })
              },
              disabled: false,
              visible: true
            },
            {
              name: '复制所有发送数据HEX到剪辑版',
              action: () => {
                let way = window.vm.List.agSelectedLine.data["方式"].toUpperCase()
                CallGoDo("socket点击右键菜单", {
                  Type: "sendHEX",
                  Theology: window.Theology,
                  IsWs: window.vm.Tabs.Request.DisplayHTTPHeader,
                  IsTCP: way.indexOf("TCP") !== -1,
                })
              },
              disabled: false,
              visible: true
            },
            {
              name: '复制所有接收数据HEX到剪辑版',
              action: () => {
                let way = window.vm.List.agSelectedLine.data["方式"].toUpperCase()
                CallGoDo("socket点击右键菜单", {
                  Type: "recHEX",
                  Theology: window.Theology,
                  IsWs: window.vm.Tabs.Request.DisplayHTTPHeader,
                  IsTCP: way.indexOf("TCP") !== -1,
                })
              },
              disabled: false,
              visible: true
            },
          ],
          disabled: false,
          visible: true
        },
        'separator',
        {
          name: '清空',
          action: () => {
            let way = window.vm.List.agSelectedLine.data["方式"].toUpperCase()
            CallGoDo("socket点击右键菜单", {
              Type: "empty",
              Theology: window.Theology,
              IsWs: window.vm.Tabs.Request.DisplayHTTPHeader,
              IsTCP: way.indexOf("TCP") !== -1
            }).then(res => {
              if (res) {
                this.Empty()
              }
            })
          },
          disabled: false,
          selected: false
        },
      ],
      RowData: [],
      columns: [
        {
          field: "#", tooltipField: '#',
          minWidth: 80, maxWidth: 80,
          menuTabs: [],
          suppressMovable: true, cellRenderer: 'imageRenderer',
          cellStyle: {'text-align': 'left'},
        },
        {
          field: "时间", tooltipField: '时间',
          minWidth: 97, maxWidth: 97,
          menuTabs: [],
          suppressMovable: true,
        },
        {
          field: "类型", tooltipField: '类型',
          minWidth: 60, maxWidth: 60,
          menuTabs: [],
          suppressMovable: true,
          cellStyle: {'text-align': 'left'}
        },
        {
          field: "长度", tooltipField: '长度',
          minWidth: 60, maxWidth: 60,
          menuTabs: [],
          suppressMovable: true,
          cellStyle: {'text-align': 'left'}
        },
        {
          field: "数据", tooltipField: '数据',
          menuTabs: [],
          suppressMovable: true,
          cellStyle: {'text-align': 'left'}
        },
      ],
      RequestId: {MessageId: -1, Theology: -1},
      ReadOnly: true,
      rowIndex: 0,
      get darkTheme() {
        return window.Theme.IsDark
      },
      set darkTheme(newValue) {
        window.Theme.IsDark = newValue
      },
    }
  },
  methods: {
    onChatScroll() {
      if (this.$refs.chatContainer) {
        this.scrollTop = this.$refs.chatContainer.scrollTop
        this.containerHeight = this.$refs.chatContainer.clientHeight
      }
    },
    isSend(msg) {
      return isSendIco(msg.ico)
    },
    truncate(str, max) {
      if (!str) return ''
      return str.length > max ? str.substring(0, max) + '...' : str
    },
    formatBytes(bytes) {
      if (bytes < 1024) return bytes + ' B'
      if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB'
      return (bytes / 1048576).toFixed(1) + ' MB'
    },
    selectMessage(idx, msg) {
      this.selectedIdx = idx
      const rowNode = this.agGridApi.getRowNode(String(msg['#']))
      if (!rowNode) {
        const allNodes = []
        this.agGridApi.forEachNode(n => {
          if (n.data && n.data['#'] === msg['#']) allNodes.push(n)
        })
        if (allNodes.length > 0) {
          allNodes[0].setSelected(true)
          this.agSelectedLine = allNodes[0]
        }
      } else {
        rowNode.setSelected(true)
        this.agSelectedLine = rowNode
      }
    },
    AddLines(objs) {
      this.agGridApi.applyTransaction({add: objs});
      if (Array.isArray(objs)) {
        this.allMessages = this.allMessages.concat(objs)
        if (this.chatMode && this.MenuItems[0].selected) {
          this.$nextTick(() => {
            if (this.$refs.chatContainer) {
              this.$refs.chatContainer.scrollTop = this.filteredMessages.length * this.itemHeight
              this.onChatScroll()
            }
          })
        }
      }
    },
    RefreshRenderedNodes() {
      const visibleRowNodes = this.agGridApi.getRenderedNodes();
      let array = []
      visibleRowNodes.forEach(node => {
        array.push(node)
      });
      this.$nextTick(() => {
        this.agGridApi.redrawRows({rowNodes: array});
      })
    },
    onGetRowStyle(params) {
      let res = {
        fontFamily: "微软雅黑"
      }
      if (params.data.background) {
        res.backgroundColor = params.data.background
      }
      return res
    },
    SetColumnsMode(ws) {
      if (ws) {
        this.columns = [
          {field: "#", tooltipField: '#', minWidth: 80, maxWidth: 80, menuTabs: [], suppressMovable: true, cellRenderer: 'imageRenderer', cellStyle: {'text-align': 'left'}},
          {field: "时间", tooltipField: '时间', minWidth: 97, maxWidth: 97, menuTabs: [], suppressMovable: true},
          {field: "类型", tooltipField: '类型', minWidth: 60, maxWidth: 60, menuTabs: [], suppressMovable: true, cellStyle: {'text-align': 'left'}},
          {field: "长度", tooltipField: '长度', minWidth: 60, maxWidth: 60, menuTabs: [], suppressMovable: true, cellStyle: {'text-align': 'left'}},
          {field: "数据", tooltipField: '数据', menuTabs: [], suppressMovable: true, cellStyle: {'text-align': 'left'}},
        ]
      } else {
        this.columns = [
          {field: "#", tooltipField: '#', minWidth: 80, maxWidth: 80, menuTabs: [], suppressMovable: true, cellRenderer: 'imageRenderer', cellStyle: {'text-align': 'left'}},
          {field: "时间", tooltipField: '时间', minWidth: 97, maxWidth: 97, menuTabs: [], suppressMovable: true},
          {field: "长度", tooltipField: '长度', minWidth: 60, maxWidth: 60, menuTabs: [], suppressMovable: true, cellStyle: {'text-align': 'left'}},
          {field: "数据", tooltipField: '数据', menuTabs: [], suppressMovable: true, cellStyle: {'text-align': 'left'}},
        ]
      }
    },
    Refresh() {
      setTimeout(() => {
        this.$nextTick(() => {
          this.agGridApi.applyTransaction({add: []});
        })
      }, 500)
    },
    Empty() {
      this.RowData = []
      this.agGridApi.setRowData(this.RowData);
      this.agSelectedLine = null
      this.allMessages = []
      this.selectedIdx = -1
    },
    onContextMenuItems() {
      let array = [];
      for (let i = 0; i < this.MenuItems.length; i++) {
        array.push(this.MenuItems[i])
      }
      return array
    },
    onRowClicked(params) {
      params.node.setSelected(true);
      this.agSelectedLine = params.node
    },
    SelectedLine(index) {
      const focusedRowNode = this.agGridApi.getRowNode(index);
      if (focusedRowNode) {
        if (this.agSelectedLine === null) {
          focusedRowNode.setSelected(true);
          this.agSelectedLine = focusedRowNode
          return
        }
        if (focusedRowNode.rowIndex !== this.agSelectedLine.rowIndex && focusedRowNode.id !== this.agSelectedLine.id) {
          focusedRowNode.setSelected(true);
          this.agSelectedLine = focusedRowNode
        }
      }
    },
    onCellFocused(event) {
      this.SelectedLine(event.rowIndex)
    },
    NewColumnsLoaded(params) {
      if (this.MenuItems[0].selected) {
        const rowCount = this.agGridApi.getDisplayedRowCount() - 1
        if (rowCount > -1) {
          this.rowIndex = rowCount
          this.agGridApi.ensureIndexVisible(rowCount)
        }
      }
    }
  },
  mounted() {
    this.agGridApi = this.$refs.agGrid.gridOptions.api
    this.$nextTick(() => {
      if (this.$refs.chatContainer) {
        this.containerHeight = this.$refs.chatContainer.clientHeight || 400
      }
    })
  }
}
</script>

<style>
.ws-toolbar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 6px;
  background: var(--ag-header-background-color, #1e1e1e);
  border-bottom: 1px solid var(--ag-border-color, #333);
  flex-shrink: 0;
}
.ws-filter-btns { display: flex; gap: 3px; }
.ws-btn {
  padding: 2px 8px;
  border: 1px solid #555;
  border-radius: 3px;
  background: transparent;
  color: #ccc;
  cursor: pointer;
  font-size: 12px;
  white-space: nowrap;
}
.ws-btn:hover { background: #333; }
.ws-btn-active { background: #2196F3 !important; color: #fff; border-color: #2196F3; }
.ws-btn-send { background: #1565c0 !important; color: #fff; border-color: #1565c0; }
.ws-btn-recv { background: #2e7d32 !important; color: #fff; border-color: #2e7d32; }
.ws-btn-toggle {
  margin-left: auto;
  font-size: 14px;
  padding: 2px 6px;
}
.ws-search {
  flex: 1;
  padding: 3px 8px;
  border: 1px solid #555;
  border-radius: 3px;
  background: #2a2a2a;
  color: #ddd;
  font-size: 12px;
  outline: none;
}
.ws-search:focus { border-color: #2196F3; }
.ws-stats {
  display: flex;
  gap: 12px;
  padding: 2px 8px;
  font-size: 11px;
  color: #999;
  background: var(--ag-header-background-color, #1e1e1e);
  border-bottom: 1px solid var(--ag-border-color, #333);
  flex-shrink: 0;
}
.ws-chat-container {
  flex: 1;
  overflow-y: auto;
  padding: 6px;
  background: var(--ag-background-color, #181818);
}
.ws-empty {
  text-align: center;
  padding: 40px;
  color: #666;
  font-size: 13px;
}
.ws-bubble-row {
  display: flex;
  margin-bottom: 4px;
  cursor: pointer;
}
.ws-bubble-row.ws-send { justify-content: flex-start; }
.ws-bubble-row.ws-recv { justify-content: flex-end; }
.ws-bubble {
  max-width: 80%;
  border-radius: 6px;
  padding: 5px 8px;
  font-size: 12px;
  line-height: 1.4;
}
.ws-send .ws-bubble {
  background: #1a3a5c;
  border: 1px solid #1565c0;
  color: #c8dff5;
}
.ws-recv .ws-bubble {
  background: #1a3c1a;
  border: 1px solid #2e7d32;
  color: #c8f5c8;
}
.ws-bubble-row.ws-selected .ws-bubble {
  box-shadow: 0 0 0 2px #ffb300;
}
.ws-bubble-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 2px;
  font-size: 11px;
}
.ws-dir-tag {
  font-weight: bold;
  font-size: 10px;
  padding: 0 4px;
  border-radius: 2px;
}
.ws-send .ws-dir-tag { background: #1565c0; color: #fff; }
.ws-recv .ws-dir-tag { background: #2e7d32; color: #fff; }
.ws-type-tag { color: #ffb300; font-weight: bold; }
.ws-len { color: #999; }
.ws-time { color: #888; margin-left: auto; }
.ws-bubble-body {
  word-break: break-all;
  white-space: pre-wrap;
  font-family: "Consolas", "微软雅黑", monospace;
  max-height: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
