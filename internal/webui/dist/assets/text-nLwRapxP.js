import{En as e,Gn as t,On as n,Qt as r,Tt as i,Yt as a,en as o,ft as s,gn as c,j as l,l as u,pt as d,u as f,wn as p}from"./client-BPPTDbz0.js";function m(e,n){return t(e,e=>{e!==void 0&&(n.value=e)}),c(()=>e.value===void 0?n.value:e.value)}function h(e,t){return c(()=>{for(let n of t)if(e[n]!==void 0)return e[n];return e[t[t.length-1]]})}var g=/^(\d|\.)+$/,_=/(\d|\.)+/;function v(e,{c:t=1,offset:n=0,attachPx:r=!0}={}){if(typeof e==`number`){let r=(e+n)*t;return r===0?`0`:`${r}px`}else if(typeof e==`string`)if(g.test(e)){let i=(Number(e)+n)*t;return r?i===0?`0`:`${i}px`:`${i}`}else{let r=_.exec(e);return r?e.replace(_,String((Number(r[0])+n)*t)):e}return e}function y(){let e=n(f,null);return e===null&&i(`use-message`,"No outer <n-message-provider /> founded. See prerequisite in https://www.naiveui.com/en-US/os-theme/components/message for more details. If you want to use `useMessage` outside setup, please check https://www.naiveui.com/zh-CN/os-theme/components/message#Q-&-A."),e}var b=a(`text`,`
 transition: color .3s var(--n-bezier);
 color: var(--n-text-color);
`,[r(`strong`,`
 font-weight: var(--n-font-weight-strong);
 `),r(`italic`,{fontStyle:`italic`}),r(`underline`,{textDecoration:`underline`}),r(`code`,`
 line-height: 1.4;
 display: inline-block;
 font-family: var(--n-font-famliy-mono);
 transition: 
 color .3s var(--n-bezier),
 border-color .3s var(--n-bezier),
 background-color .3s var(--n-bezier);
 box-sizing: border-box;
 padding: .05em .35em 0 .35em;
 border-radius: var(--n-code-border-radius);
 font-size: .9em;
 color: var(--n-code-text-color);
 background-color: var(--n-code-color);
 border: var(--n-code-border);
 `)]),x=p({name:`Text`,props:Object.assign(Object.assign({},l.props),{code:Boolean,type:{type:String,default:`default`},delete:Boolean,strong:Boolean,italic:Boolean,underline:Boolean,depth:[String,Number],tag:String,as:{type:String,validator:()=>!0,default:void 0}}),setup(e){let{mergedClsPrefixRef:t,inlineThemeDisabled:n}=d(e),r=l(`Typography`,`-text`,b,u,e,t),i=c(()=>{let{depth:t,type:n}=e,i=n==="default"?t===void 0?`textColor`:`textColor${t}Depth`:o(`textColor`,n),{common:{fontWeightStrong:a,fontFamilyMono:s,cubicBezierEaseInOut:c},self:{codeTextColor:l,codeBorderRadius:u,codeColor:d,codeBorder:f,[i]:p}}=r.value;return{"--n-bezier":c,"--n-text-color":p,"--n-font-weight-strong":a,"--n-font-famliy-mono":s,"--n-code-border-radius":u,"--n-code-text-color":l,"--n-code-color":d,"--n-code-border":f}}),a=n?s(`text`,c(()=>`${e.type[0]}${e.depth||``}`),i,e):void 0;return{mergedClsPrefix:t,compitableTag:h(e,[`as`,`tag`]),cssVars:n?void 0:i,themeClass:a?.themeClass,onRender:a?.onRender}},render(){var t,n;let{mergedClsPrefix:r}=this;(t=this.onRender)==null||t.call(this);let i=[`${r}-text`,this.themeClass,{[`${r}-text--code`]:this.code,[`${r}-text--delete`]:this.delete,[`${r}-text--strong`]:this.strong,[`${r}-text--italic`]:this.italic,[`${r}-text--underline`]:this.underline}],a=(n=this.$slots).default?.call(n);return this.code?e(`code`,{class:i,style:this.cssVars},this.delete?e(`del`,null,a):a):this.delete?e(`del`,{class:i,style:this.cssVars},a):e(this.compitableTag||`span`,{class:i,style:this.cssVars},a)}});export{m as a,h as i,y as n,v as r,x as t};