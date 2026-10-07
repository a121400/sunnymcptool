<script>
//

export default {
  props: ['params'],
  watch: {
    "value"(m, l) {
      this.Params.data['替换类型'] = m
      this.Params.api.applyTransaction({update: [this.Params.data]});
    },
  },
  data() {
    return {
      Params: null,
      value: "",
    }
  },
  created() {
    this.Params = this.params;
    this.value = this.Params.data['替换类型']
  },
  mounted() {

  }, methods: {}
};
</script>
<template>
  <el-select v-model="value" class="m-2" placeholder="Select" size="small"
             style="position: relative;top: -4px;left: -10px; width: 105px">
    <el-tooltip class="item" effect="dark" content="如果选择此项,那么 源内容及替换内容 都要求填写十六进制字符串"
                placement="top">
      <el-option key="HEX" label="HEX" value="HEX"/>
    </el-tooltip>
    <el-tooltip class="item" effect="dark" content="如果选择此项,那么 源内容及替换内容 都要求填写Base64字符串"
                placement="top">
      <el-option key="Base64" label="Base64" value="Base64"/>
    </el-tooltip>
    <el-tooltip class="item" effect="dark" content="如果选择此项,那么 源内容及替换内容 中的中文字符串都按UTF8编码处理"
                placement="top">
      <el-option key="String(UTF8)" label="String(UTF8)" value="String(UTF8)"/>
    </el-tooltip>
    <el-tooltip class="item" effect="dark" content="如果选择此项,那么 源内容及替换内容 中的中文字符串都按GBK编码处理"
                placement="top">
      <el-option key="String(GBK)" label="String(GBK)" value="String(GBK)"/>
    </el-tooltip>
    <el-tooltip class="item" effect="dark"
                content="源内容填写 URL 里要出现的文字，替换内容填写本地文件绝对路径。文件每次请求都会重新读取"
                placement="top">
      <el-option key="响应文件" label="响应文件" value="响应文件"/>
    </el-tooltip>
    <el-tooltip class="item" effect="dark"
                content="源内容填头名则整段改成替换内容，留空表示删除。也可写 头名||要替换的片段"
                placement="top">
      <el-option key="请求头" label="请求头" value="请求头"/>
    </el-tooltip>
    <el-tooltip class="item" effect="dark"
                content="源内容填头名则整段改成替换内容，留空表示删除。也可写 头名||要替换的片段"
                placement="top">
      <el-option key="响应头" label="响应头" value="响应头"/>
    </el-tooltip>
    <el-tooltip class="item" effect="dark" content="只替换请求正文，不改 URL 和协议头" placement="top">
      <el-option key="请求体" label="请求体" value="请求体"/>
    </el-tooltip>
    <el-tooltip class="item" effect="dark" content="只替换响应正文，不改 URL 和协议头" placement="top">
      <el-option key="响应体" label="响应体" value="响应体"/>
    </el-tooltip>
    <el-tooltip class="item" effect="dark" content="源内容是正则，替换内容可用 $1。只作用于请求正文" placement="top">
      <el-option key="正则请求" label="正则请求" value="正则请求"/>
    </el-tooltip>
    <el-tooltip class="item" effect="dark" content="源内容是正则，替换内容可用 $1。只作用于响应正文" placement="top">
      <el-option key="正则响应" label="正则响应" value="正则响应"/>
    </el-tooltip>
    <el-tooltip class="item" effect="dark" content="只替换 URL，不改协议头和正文" placement="top">
      <el-option key="仅URL" label="仅URL" value="仅URL"/>
    </el-tooltip>

  </el-select>
</template>
