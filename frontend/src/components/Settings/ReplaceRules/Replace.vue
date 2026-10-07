<template>
  <ag-grid-vue
      ref="agGrid"
      style="height: 100%;"
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
</template>

<script>
import '@ag-grid-community/styles/ag-grid.css';
import '@ag-grid-community/styles/ag-theme-balham.css';
import {AgGridVue} from '@ag-grid-community/vue3';
import {ClipboardModule} from '@ag-grid-enterprise/clipboard';
import {SetFilterModule} from '@ag-grid-enterprise/set-filter';
import {ExcelExportModule} from '@ag-grid-enterprise/excel-export';
import ReplaceType from "./ReplaceType.vue";
import {CallGoDo} from "../../CallbackEventsOn.js";
import {ElMessage} from "element-plus";

export default {
  props: ['readOnly'],
  watch: {
    readOnly(value) {
      this.ReadOnly = value
    },
  },
  components: {
    'ag-grid-vue': AgGridVue, replaceType: ReplaceType,
  },
  computed: {
    LineSelected() {
      if (this.ReadOnly) {
        this.MenuItems[0].visible = false
        this.MenuItems[1].visible = false
        this.columns[0].editable = false
        this.columns[1].editable = false
      } else {
        this.MenuItems[1].visible = this.agSelectedLine !== null;
        this.MenuItems[0].visible = true
        this.columns[0].editable = true
        this.columns[1].editable = true
      }
      //务必返回 false
      return false
    }
  },
  data() {
    return {
      //是否被修改
      IsHasModify: false,
      //当前选中行
      agSelectedLine: null,
      overlayNoRowsTemplate: `<span style="padding: 20px;" id="HookMessageText">无替换规则</span>`,
      leftModules: [SetFilterModule, ClipboardModule],
      rightModules: [ExcelExportModule],
      gridOptions: {
        //rowSelection: 'multiple',
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
        // 禁用自动滚动到第一行
        suppressScrollOnNewData: true,
      },
      defaultColDef: {
        flex: 1,
        minWidth: 10,
        // 禁用全部列的排序功能
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
          name: '增加',
          action: () => {
            this.AddLine("String(UTF8)", "baidu.com", "www.baidu.com", "未保存")
            this.IsHasModify = true
          },
          disabled: false,
          visible: true
        },
        {
          name: '删除',
          action: () => {
            if (this.agSelectedLine !== null) {
              this.RowData.splice(this.agSelectedLine.rowIndex, 1)
              this.$nextTick(() => {
                this.agGridApi.setRowData(this.RowData);
                this.agSelectedLine = null
                this.IsHasModify = true
              })
            }
          },
          disabled: false,
          visible: false
        },
        "separator",
        {
          name: '保存变更',
          action: () => {
            CallGoDo("保存替换规则", {Data: this.RowData}).then(res => {
              let mm = 0
              for (let i = 0; i < this.RowData.length; i++) {
                let ok = false
                if (res) {
                  for (let n = 0; n < res.length; n++) {
                    if (res[n] === this.RowData[i].Hash) {
                      ok = true
                      break
                    }
                  }
                }
                if (!ok) {
                  this.RowData[i].状态 = "已保存"
                } else {
                  this.RowData[i].状态 = "保存失败"
                  mm++
                }
              }
              this.agGridApi.setRowData(this.RowData);
              if (mm > 0) {
                ElMessage({
                  message: '有 ' + mm + " 项保存失败,请检查数据是否正确！",
                  type: 'warning',
                })
              } else {
                ElMessage({
                  message: '已保存',
                  type: 'success',
                })
              }
            })
          },
          disabled:
              false,
          visible:
              true
        }
        ,
      ],
      RowData: [],
      columns:
          [
            {
              field: "替换类型", tooltipField: '替换类型',
              minWidth: 108,
              maxWidth: 108,
              // 不显示过滤器
              menuTabs: [],
              //禁止列拖动
              suppressMovable: true,
              editable: true,
              sortable: true,
              cellRenderer: 'replaceType',
              cellStyle: {'text-align': 'left'},
            },
            {
              field: "源内容", tooltipField: '源内容',
              minWidth: 80,
              // 不显示过滤器
              menuTabs: [],
              //禁止列拖动
              suppressMovable: true,
              editable: true,
              cellStyle: {'text-align': 'left'}
            },
            {
              field: "替换内容", tooltipField: '替换内容',
              minWidth: 80,
              // 不显示过滤器
              menuTabs: [],
              //禁止列拖动
              suppressMovable: true,
              editable: true,
              cellStyle: {'text-align': 'left'}
            },
            {
              field: "生效范围", tooltipField: '生效范围',
              headerTooltip: '留空表示全部流量。填写后只对 URL 包含这段文字的请求生效。以 re: 开头则按正则匹配完整 URL',
              minWidth: 90,
              menuTabs: [],
              suppressMovable: true,
              editable: true,
              cellStyle: {'text-align': 'left'}
            },
            {
              field: "状态", tooltipField: '状态',
              minWidth: 100,
              maxWidth: 100,
              // 不显示过滤器
              menuTabs: [],
              //禁止列拖动
              suppressMovable: true,
              editable: true,
              cellStyle: {'text-align': 'left'}
            },
          ],
      RequestId:
          {
            MessageId: -1, Theology:
                -1
          }
      ,
      ReadOnly: false,
    }
  }
  ,
  methods: {
    AddLine(type, a1, a2, a3) {
      let obj = {}
      obj["替换类型"] = type
      obj["源内容"] = a1
      obj["替换内容"] = a2
      obj["生效范围"] = ""
      obj["状态"] = a3
      obj.Hash = (new Date().getTime()) + "";
      this.RowData.push(obj)
      this.$nextTick(() => {
        this.agGridApi.setRowData(this.RowData);
      })
      this.IsHasModify = false
    },
    AddLines(objs) {
      const array = []
      for (let i = 0; i < objs.length; i++) {
        const Type = objs[i]['Type']
        const Src = objs[i]['Src']
        const Dest = objs[i]['Dest']
        let obj = {}
        obj["替换类型"] = Type
        obj["源内容"] = Src
        obj["替换内容"] = Dest
        obj["生效范围"] = objs[i]['Scope'] || ""
        obj["状态"] = "已保存"
        obj.Hash = objs[i]['Hash'] || ((new Date().getTime()) + "" + i);
        array.push(obj)
      }
      this.RowData = array
      this.$nextTick(() => {
        this.agGridApi.setRowData(this.RowData);
      })
      this.IsHasModify = false
    },
    Empty() {
      this.RowData = []
      this.agGridApi.setRowData(this.RowData);
      this.agSelectedLine = null
    },
    onContextMenuItems() {
      let array = [];
      for (let i = 0; i < this.MenuItems.length; i++) {
        if (this.MenuItems[i].visible !== false) {
          array.push(this.MenuItems[i])
        }
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
    handleDocumentClick(event) {
      try {
        const editingCells = this.agGridApi.getEditingCells();
        if (editingCells.length > 0) {
          if (event.target) {
            const mm = event.target
            const ma = this.$refs.agGrid.$el.getElementsByClassName("ag-input-field-input ag-text-field-input")
            for (let i = 0; i < ma.length; i++) {
              if (ma[i] === mm) {
                return
              }
            }
          }
          this.agGridApi.stopEditing(); // 退出单元格编辑
        }
      } catch (e) {
        console.log(e)
      }
    },
  },
  mounted() {
    this.agGridApi = this.$refs.agGrid.gridOptions.api
    document.addEventListener('click', this.handleDocumentClick);
  }
}
</script>

<style>

</style>