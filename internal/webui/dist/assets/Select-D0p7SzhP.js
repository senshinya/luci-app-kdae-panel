import{$t as e,An as t,At as n,C as r,En as i,Fn as a,Ft as o,Gn as s,Gt as c,Ht as l,In as u,Jn as d,Jt as f,Kn as p,Kt as m,Mn as h,On as g,Ot as _,P as v,Pn as y,Pt as b,Qt as x,Wt as S,Yt as C,Zt as w,_ as T,an as E,cn as D,dt as ee,en as O,fn as k,ft as te,gn as A,gt as j,j as M,jn as N,k as P,kt as F,pt as ne,rr as I,sr as L,wn as R,wt as z,xt as B,yt as re,zn as V}from"./client-CHTsrZM3.js";import{t as ie}from"./next-frame-once-qdYFoq8G.js";import{i as H,n as U,r as W,t as G}from"./create-CXb85VLd.js";import{l as K}from"./light-FpSYQy6K.js";import{a as q,c as ae,d as oe,i as J,l as Y,o as se,s as X,t as Z,u as ce}from"./Popover-Ce84eOiN.js";import{t as le}from"./use-merged-state-DSdsnVdt.js";import{i as ue}from"./text-G-kXxcoP.js";import{t as de}from"./use-locale-DChWZDhp.js";import{n as Q}from"./Input-nrkBfoWi.js";import{t as fe}from"./Empty-Dp7MBd92.js";import{a as pe,r as me,t as he}from"./light-CSJxjZi1.js";import{n as ge}from"./Icon-DGOuQrf2.js";import{E as _e,P as ve,j as $}from"./index-Dk2u75Jj.js";function ye(e){return e&-e}var be=class{constructor(e,t){this.l=e,this.min=t;let n=Array(e+1);for(let t=0;t<e+1;++t)n[t]=0;this.ft=n}add(e,t){if(t===0)return;let{l:n,ft:r}=this;for(e+=1;e<=n;)r[e]+=t,e+=ye(e)}get(e){return this.sum(e+1)-this.sum(e)}sum(e){if(e===void 0&&(e=this.l),e<=0)return 0;let{ft:t,min:n,l:r}=this;if(e>r)throw Error("[FinweckTree.sum]: `i` is larger than length.");let i=e*n;for(;e>0;)i+=t[e],e-=ye(e);return i}getBound(e){let t=0,n=this.l;for(;n>t;){let r=Math.floor((t+n)/2),i=this.sum(r);if(i>e){n=r;continue}else if(i<e){if(t===r)return this.sum(t+1)<=e?t+1:r;t=r}else return r}return t}},xe;function Se(){return typeof document>`u`?!1:(xe===void 0&&(xe=`matchMedia`in window&&window.matchMedia(`(pointer:coarse)`).matches),xe)}var Ce;function we(){return typeof document>`u`?1:(Ce===void 0&&(Ce=`chrome`in window?window.devicePixelRatio:1),Ce)}var Te=`VVirtualListXScroll`;function Ee({columnsRef:e,renderColRef:t,renderItemWithColsRef:n}){let r=I(0),i=I(0),a=A(()=>{let t=e.value;if(t.length===0)return null;let n=new be(t.length,0);return t.forEach((e,t)=>{n.add(t,e.width)}),n});return V(Te,{startIndexRef:o(()=>{let e=a.value;return e===null?0:Math.max(e.getBound(i.value)-1,0)}),endIndexRef:o(()=>{let t=a.value;return t===null?0:Math.min(t.getBound(i.value+r.value)+1,e.value.length-1)}),columnsRef:e,renderColRef:t,renderItemWithColsRef:n,getLeft:e=>{let t=a.value;return t===null?0:t.sum(e)}}),{listWidthRef:r,scrollLeftRef:i}}var De=R({name:`VirtualListRow`,props:{index:{type:Number,required:!0},item:{type:Object,required:!0}},setup(){let{startIndexRef:e,endIndexRef:t,columnsRef:n,getLeft:r,renderColRef:i,renderItemWithColsRef:a}=g(Te);return{startIndex:e,endIndex:t,columns:n,renderCol:i,renderItemWithCols:a,getLeft:r}},render(){let{startIndex:e,endIndex:t,columns:n,renderCol:r,renderItemWithCols:i,getLeft:a,item:o}=this;if(i!=null)return i({itemIndex:this.index,startColIndex:e,endColIndex:t,allColumns:n,item:o,getLeft:a});if(r!=null){let i=[];for(let s=e;s<=t;++s){let e=n[s];i.push(r({column:e,left:a(s),item:o}))}return i}return null}}),Oe=q(`.v-vl`,{maxHeight:`inherit`,height:`100%`,overflow:`auto`,minWidth:`1px`},[q(`&:not(.v-vl--show-scrollbar)`,{scrollbarWidth:`none`},[q(`&::-webkit-scrollbar, &::-webkit-scrollbar-track-piece, &::-webkit-scrollbar-thumb`,{width:0,height:0,display:`none`})])]),ke=R({name:`VirtualList`,inheritAttrs:!1,props:{showScrollbar:{type:Boolean,default:!0},columns:{type:Array,default:()=>[]},renderCol:Function,renderItemWithCols:Function,items:{type:Array,default:()=>[]},itemSize:{type:Number,required:!0},itemResizable:Boolean,itemsStyle:[String,Object],visibleItemsTag:{type:[String,Object],default:`div`},visibleItemsProps:Object,ignoreItemResize:Boolean,onScroll:Function,onWheel:Function,onResize:Function,defaultScrollKey:[Number,String],defaultScrollIndex:Number,keyField:{type:String,default:`key`},paddingTop:{type:[Number,String],default:0},paddingBottom:{type:[Number,String],default:0}},setup(e){let t=n();Oe.mount({id:`vueuc/virtual-list`,head:!0,anchorMetaName:se,ssr:t}),u(()=>{let{defaultScrollIndex:t,defaultScrollKey:n}=e;t==null?n!=null&&C({key:n}):C({index:t})});let r=!1,i=!1;h(()=>{if(r=!1,!i){i=!0;return}C({top:b.value,left:f.value})}),a(()=>{r=!0,i||=!0});let s=o(()=>{if(e.renderCol==null&&e.renderItemWithCols==null||e.columns.length===0)return;let t=0;return e.columns.forEach(e=>{t+=e.width}),t}),d=A(()=>{let t=new Map,{keyField:n}=e;return e.items.forEach((e,r)=>{t.set(e[n],r)}),t}),{scrollLeftRef:f,listWidthRef:p}=Ee({columnsRef:L(e,`columns`),renderColRef:L(e,`renderCol`),renderItemWithColsRef:L(e,`renderItemWithCols`)}),m=I(null),g=I(void 0),_=new Map,v=A(()=>{let{items:t,itemSize:n,keyField:r}=e,i=new be(t.length,n);return t.forEach((e,t)=>{let n=e[r],a=_.get(n);a!==void 0&&i.add(t,a)}),i}),y=I(0),b=I(0),x=o(()=>Math.max(v.value.getBound(b.value-l(e.paddingTop))-1,0)),S=A(()=>{let{value:t}=g;if(t===void 0)return[];let{items:n,itemSize:r}=e,i=x.value,a=Math.min(i+Math.ceil(t/r+1),n.length-1),o=[];for(let e=i;e<=a;++e)o.push(n[e]);return o}),C=(e,t)=>{if(typeof e==`number`){D(e,t,`auto`);return}let{left:n,top:r,index:i,key:a,position:o,behavior:s,debounce:c=!0}=e;if(n!==void 0||r!==void 0)D(n,r,s);else if(i!==void 0)E(i,s,c);else if(a!==void 0){let e=d.value.get(a);e!==void 0&&E(e,s,c)}else o===`bottom`?D(0,2**53-1,s):o===`top`&&D(0,0,s)},w,T=null;function E(t,n,r){let{value:i}=v,a=i.sum(t)+l(e.paddingTop);if(!r)m.value.scrollTo({left:0,top:a,behavior:n});else{w=t,T!==null&&window.clearTimeout(T),T=window.setTimeout(()=>{w=void 0,T=null},16);let{scrollTop:e,offsetHeight:r}=m.value;if(a>e){let o=i.get(t);a+o<=e+r||m.value.scrollTo({left:0,top:a+o-r,behavior:n})}else m.value.scrollTo({left:0,top:a,behavior:n})}}function D(e,t,n){m.value.scrollTo({left:e,top:t,behavior:n})}function ee(t,n){if(r||e.ignoreItemResize||P(n.target))return;let{value:i}=v,a=d.value.get(t),o=i.get(a),s=n.borderBoxSize?.[0]?.blockSize??n.contentRect.height;if(s===o)return;s-e.itemSize===0?_.delete(t):_.set(t,s-e.itemSize);let c=s-o;if(c===0)return;i.add(a,c);let l=m.value;if(l!=null){if(w===void 0){let e=i.sum(a);l.scrollTop>e&&l.scrollBy(0,c)}else(a<w||a===w&&s+i.sum(a)>l.scrollTop+l.offsetHeight)&&l.scrollBy(0,c);N()}y.value++}let O=!Se(),k=!1;function te(t){var n;(n=e.onScroll)==null||n.call(e,t),(!O||!k)&&N()}function j(t){var n;if((n=e.onWheel)==null||n.call(e,t),O){let e=m.value;if(e!=null){if(t.deltaX===0&&(e.scrollTop===0&&t.deltaY<=0||e.scrollTop+e.offsetHeight>=e.scrollHeight&&t.deltaY>=0))return;t.preventDefault(),e.scrollTop+=t.deltaY/we(),e.scrollLeft+=t.deltaX/we(),N(),k=!0,ie(()=>{k=!1})}}}function M(t){if(r||P(t.target))return;if(e.renderCol==null&&e.renderItemWithCols==null){if(t.contentRect.height===g.value)return}else if(t.contentRect.height===g.value&&t.contentRect.width===p.value)return;g.value=t.contentRect.height,p.value=t.contentRect.width;let{onResize:n}=e;n!==void 0&&n(t)}function N(){let{value:e}=m;e!=null&&(b.value=e.scrollTop,f.value=e.scrollLeft)}function P(e){let t=e;for(;t!==null;){if(t.style.display===`none`)return!0;t=t.parentElement}return!1}return{listHeight:g,listStyle:{overflow:`auto`},keyToIndex:d,itemsStyle:A(()=>{let{itemResizable:t}=e,n=c(v.value.sum());return y.value,[e.itemsStyle,{boxSizing:`content-box`,width:c(s.value),height:t?``:n,minHeight:t?n:``,paddingTop:c(e.paddingTop),paddingBottom:c(e.paddingBottom)}]}),visibleItemsStyle:A(()=>(y.value,{transform:`translateY(${c(v.value.sum(x.value))})`})),viewportItems:S,listElRef:m,itemsElRef:I(null),scrollTo:C,handleListResize:M,handleListScroll:te,handleListWheel:j,handleItemResize:ee}},render(){let{itemResizable:e,keyField:n,keyToIndex:r,visibleItemsTag:a}=this;return i(_,{onResize:this.handleListResize},{default:()=>{var o;return i(`div`,t(this.$attrs,{class:[`v-vl`,this.showScrollbar&&`v-vl--show-scrollbar`],onScroll:this.handleListScroll,onWheel:this.handleListWheel,ref:`listElRef`}),[this.items.length===0?(o=this.$slots).empty?.call(o):i(`div`,{ref:`itemsElRef`,class:`v-vl-items`,style:this.itemsStyle},[i(a,Object.assign({class:`v-vl-visible-items`,style:this.visibleItemsStyle},this.visibleItemsProps),{default:()=>{let{renderCol:t,renderItemWithCols:a}=this;return this.viewportItems.map(o=>{let s=o[n],c=r.get(s),l=t==null?void 0:i(De,{index:c,item:o}),u=a==null?void 0:i(De,{index:c,item:o}),d=this.$slots.default({item:o,renderedCols:l,renderedItemWithCols:u,index:c})[0];return e?i(_,{key:s,onResize:e=>this.handleItemResize(s,e)},{default:()=>d}):(d.key=s,d)})}})])])}})}});function Ae(e,t){t&&(u(()=>{let{value:n}=e;n&&F.registerHandler(n,t)}),s(e,(e,t)=>{t&&F.unregisterHandler(t)},{deep:!1}),y(()=>{let{value:t}=e;t&&F.unregisterHandler(t)}))}function je(e){switch(typeof e){case`string`:return e||void 0;case`number`:return String(e);default:return}}function Me(e){let t=e.filter(e=>e!==void 0);if(t.length!==0)return t.length===1?t[0]:t=>{e.forEach(e=>{e&&e(t)})}}var Ne=R({name:`Checkmark`,render(){return i(`svg`,{xmlns:`http://www.w3.org/2000/svg`,viewBox:`0 0 16 16`},i(`g`,{fill:`none`},i(`path`,{d:`M14.046 3.486a.75.75 0 0 1-.032 1.06l-7.93 7.474a.85.85 0 0 1-1.188-.022l-2.68-2.72a.75.75 0 1 1 1.068-1.053l2.234 2.267l7.468-7.038a.75.75 0 0 1 1.06.032z`,fill:`currentColor`})))}}),Pe=R({props:{onFocus:Function,onBlur:Function},setup(e){return()=>i(`div`,{style:`width: 0; height: 0`,tabindex:0,onFocus:e.onFocus,onBlur:e.onBlur})}}),Fe=R({name:`NBaseSelectGroupHeader`,props:{clsPrefix:{type:String,required:!0},tmNode:{type:Object,required:!0}},setup(){let{renderLabelRef:e,renderOptionRef:t,labelFieldRef:n,nodePropsRef:r}=g(oe);return{labelField:n,nodeProps:r,renderLabel:e,renderOption:t}},render(){let{clsPrefix:e,renderLabel:t,renderOption:n,nodeProps:r,tmNode:{rawNode:a}}=this,o=r?.(a),s=t?t(a,!1):$(a[this.labelField],a,!1),c=i(`div`,Object.assign({},o,{class:[`${e}-base-select-group-header`,o?.class]}),s);return a.render?a.render({node:c,option:a}):n?n({node:c,option:a,selected:!1}):c}});function Ie(e,t){return i(E,{name:`fade-in-scale-up-transition`},{default:()=>e?i(P,{clsPrefix:t,class:`${t}-base-select-option__check`},{default:()=>i(Ne)}):null})}var Le=R({name:`NBaseSelectOption`,props:{clsPrefix:{type:String,required:!0},tmNode:{type:Object,required:!0}},setup(e){let{valueRef:t,pendingTmNodeRef:n,multipleRef:r,valueSetRef:i,renderLabelRef:a,renderOptionRef:s,labelFieldRef:c,valueFieldRef:l,showCheckmarkRef:u,nodePropsRef:d,handleOptionClick:f,handleOptionMouseEnter:p}=g(oe),m=o(()=>{let{value:t}=n;return t?e.tmNode.key===t.key:!1});function h(t){let{tmNode:n}=e;n.disabled||f(t,n)}function _(t){let{tmNode:n}=e;n.disabled||p(t,n)}function v(t){let{tmNode:n}=e,{value:r}=m;n.disabled||r||p(t,n)}return{multiple:r,isGrouped:o(()=>{let{tmNode:t}=e,{parent:n}=t;return n&&n.rawNode.type===`group`}),showCheckmark:u,nodeProps:d,isPending:m,isSelected:o(()=>{let{value:n}=t,{value:a}=r;if(n===null)return!1;let o=e.tmNode.rawNode[l.value];if(a){let{value:e}=i;return e.has(o)}else return n===o}),labelField:c,renderLabel:a,renderOption:s,handleMouseMove:v,handleMouseEnter:_,handleClick:h}},render(){let{clsPrefix:e,tmNode:{rawNode:t},isSelected:n,isPending:r,isGrouped:a,showCheckmark:o,nodeProps:s,renderOption:c,renderLabel:l,handleClick:u,handleMouseEnter:d,handleMouseMove:f}=this,p=Ie(n,e),m=l?[l(t,n),o&&p]:[$(t[this.labelField],t,n),o&&p],h=s?.(t),g=i(`div`,Object.assign({},h,{class:[`${e}-base-select-option`,t.class,h?.class,{[`${e}-base-select-option--disabled`]:t.disabled,[`${e}-base-select-option--selected`]:n,[`${e}-base-select-option--grouped`]:a,[`${e}-base-select-option--pending`]:r,[`${e}-base-select-option--show-checkmark`]:o}],style:[h?.style||``,t.style||``],onClick:Me([u,h?.onClick]),onMouseenter:Me([d,h?.onMouseenter]),onMousemove:Me([f,h?.onMousemove])}),i(`div`,{class:`${e}-base-select-option__content`},m));return t.render?t.render({node:g,option:t,selected:n}):c?c({node:g,option:t,selected:n}):g}}),Re=C(`base-select-menu`,`
 line-height: 1.5;
 outline: none;
 z-index: 0;
 position: relative;
 border-radius: var(--n-border-radius);
 transition:
 background-color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier);
 background-color: var(--n-color);
`,[C(`scrollbar`,`
 max-height: var(--n-height);
 `),C(`virtual-list`,`
 max-height: var(--n-height);
 `),C(`base-select-option`,`
 min-height: var(--n-option-height);
 font-size: var(--n-option-font-size);
 display: flex;
 align-items: center;
 `,[w(`content`,`
 z-index: 1;
 white-space: nowrap;
 text-overflow: ellipsis;
 overflow: hidden;
 `)]),C(`base-select-group-header`,`
 min-height: var(--n-option-height);
 font-size: .93em;
 display: flex;
 align-items: center;
 `),C(`base-select-menu-option-wrapper`,`
 position: relative;
 width: 100%;
 `),w(`loading, empty`,`
 display: flex;
 padding: 12px 32px;
 flex: 1;
 justify-content: center;
 `),w(`loading`,`
 color: var(--n-loading-color);
 font-size: var(--n-loading-size);
 `),w(`header`,`
 padding: 8px var(--n-option-padding-left);
 font-size: var(--n-option-font-size);
 transition: 
 color .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 border-bottom: 1px solid var(--n-action-divider-color);
 color: var(--n-action-text-color);
 `),w(`action`,`
 padding: 8px var(--n-option-padding-left);
 font-size: var(--n-option-font-size);
 transition: 
 color .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 border-top: 1px solid var(--n-action-divider-color);
 color: var(--n-action-text-color);
 `),C(`base-select-group-header`,`
 position: relative;
 cursor: default;
 padding: var(--n-option-padding);
 color: var(--n-group-header-text-color);
 `),C(`base-select-option`,`
 cursor: pointer;
 position: relative;
 padding: var(--n-option-padding);
 transition:
 color .3s var(--n-bezier),
 opacity .3s var(--n-bezier);
 box-sizing: border-box;
 color: var(--n-option-text-color);
 opacity: 1;
 `,[x(`show-checkmark`,`
 padding-right: calc(var(--n-option-padding-right) + 20px);
 `),f(`&::before`,`
 content: "";
 position: absolute;
 left: 4px;
 right: 4px;
 top: 0;
 bottom: 0;
 border-radius: var(--n-border-radius);
 transition: background-color .3s var(--n-bezier);
 `),f(`&:active`,`
 color: var(--n-option-text-color-pressed);
 `),x(`grouped`,`
 padding-left: calc(var(--n-option-padding-left) * 1.5);
 `),x(`pending`,[f(`&::before`,`
 background-color: var(--n-option-color-pending);
 `)]),x(`selected`,`
 color: var(--n-option-text-color-active);
 `,[f(`&::before`,`
 background-color: var(--n-option-color-active);
 `),x(`pending`,[f(`&::before`,`
 background-color: var(--n-option-color-active-pending);
 `)])]),x(`disabled`,`
 cursor: not-allowed;
 `,[e(`selected`,`
 color: var(--n-option-text-color-disabled);
 `),x(`selected`,`
 opacity: var(--n-option-opacity-disabled);
 `)]),w(`check`,`
 font-size: 16px;
 position: absolute;
 right: calc(var(--n-option-padding-right) - 4px);
 top: calc(50% - 7px);
 color: var(--n-option-check-color);
 transition: color .3s var(--n-bezier);
 `,[_e({enterScale:`0.5`})])])]),ze=R({name:`InternalSelectMenu`,props:Object.assign(Object.assign({},M.props),{clsPrefix:{type:String,required:!0},scrollable:{type:Boolean,default:!0},treeMate:{type:Object,required:!0},multiple:Boolean,size:{type:String,default:`medium`},value:{type:[String,Number,Array],default:null},autoPending:Boolean,virtualScroll:{type:Boolean,default:!0},show:{type:Boolean,default:!0},labelField:{type:String,default:`label`},valueField:{type:String,default:`value`},loading:Boolean,focusable:Boolean,renderLabel:Function,renderOption:Function,nodeProps:Function,showCheckmark:{type:Boolean,default:!0},onMousedown:Function,onScroll:Function,onFocus:Function,onBlur:Function,onKeyup:Function,onKeydown:Function,onTabOut:Function,onMouseenter:Function,onMouseleave:Function,onResize:Function,resetMenuOnOptionsChange:{type:Boolean,default:!0},inlineThemeDisabled:Boolean,scrollbarProps:Object,onToggle:Function}),setup(e){let{mergedClsPrefixRef:t,mergedRtlRef:n,mergedComponentPropsRef:r}=ne(e),i=v(`InternalSelectMenu`,n,t),a=M(`InternalSelectMenu`,`-internal-select-menu`,Re,pe,e,L(e,`clsPrefix`)),o=I(null),c=I(null),d=I(null),f=A(()=>e.treeMate.getFlattenedNodes()),p=A(()=>U(f.value)),m=I(null);function h(){let{treeMate:t}=e,n=null,{value:r}=e;r===null?n=t.getFirstAvailableNode():(n=e.multiple?t.getNode((r||[])[(r||[]).length-1]):t.getNode(r),(!n||n.disabled)&&(n=t.getFirstAvailableNode())),W(n||null)}function g(){let{value:t}=m;t&&!e.treeMate.getNode(t.key)&&(m.value=null)}let _;s(()=>e.show,t=>{t?_=s(()=>e.treeMate,()=>{e.resetMenuOnOptionsChange?(e.autoPending?h():g(),N(G)):g()},{immediate:!0}):_?.()},{immediate:!0}),y(()=>{_?.()});let b=A(()=>l(a.value.self[O(`optionHeight`,e.size)])),x=A(()=>S(a.value.self[O(`padding`,e.size)])),C=A(()=>e.multiple&&Array.isArray(e.value)?new Set(e.value):new Set),w=A(()=>{let e=f.value;return e&&e.length===0}),T=A(()=>r?.value?.Select?.renderEmpty);function E(t){let{onToggle:n}=e;n&&n(t)}function D(t){let{onScroll:n}=e;n&&n(t)}function ee(e){var t;(t=d.value)==null||t.sync(),D(e)}function k(){var e;(e=d.value)==null||e.sync()}function j(){let{value:e}=m;return e||null}function P(e,t){t.disabled||W(t,!1)}function F(e,t){t.disabled||E(t)}function R(t){var n;H(t,`action`)||(n=e.onKeyup)==null||n.call(e,t)}function z(t){var n;H(t,`action`)||(n=e.onKeydown)==null||n.call(e,t)}function B(t){var n;(n=e.onMousedown)==null||n.call(e,t),!e.focusable&&t.preventDefault()}function re(){let{value:e}=m;e&&W(e.getNext({loop:!0}),!0)}function ie(){let{value:e}=m;e&&W(e.getPrev({loop:!0}),!0)}function W(e,t=!1){m.value=e,t&&G()}function G(){var t,n;let r=m.value;if(!r)return;let i=p.value(r.key);i!==null&&(e.virtualScroll?(t=c.value)==null||t.scrollTo({index:i}):(n=d.value)==null||n.scrollTo({index:i,elSize:b.value}))}function K(t){var n;o.value?.contains(t.target)&&((n=e.onFocus)==null||n.call(e,t))}function q(t){var n;o.value?.contains(t.relatedTarget)||(n=e.onBlur)==null||n.call(e,t)}V(oe,{handleOptionMouseEnter:P,handleOptionClick:F,valueSetRef:C,pendingTmNodeRef:m,nodePropsRef:L(e,`nodeProps`),showCheckmarkRef:L(e,`showCheckmark`),multipleRef:L(e,`multiple`),valueRef:L(e,`value`),renderLabelRef:L(e,`renderLabel`),renderOptionRef:L(e,`renderOption`),labelFieldRef:L(e,`labelField`),valueFieldRef:L(e,`valueField`)}),V(ce,o),u(()=>{let{value:e}=d;e&&e.sync()});let ae=A(()=>{let{size:t}=e,{common:{cubicBezierEaseInOut:n},self:{height:r,borderRadius:i,color:o,groupHeaderTextColor:s,actionDividerColor:c,optionTextColorPressed:l,optionTextColor:u,optionTextColorDisabled:d,optionTextColorActive:f,optionOpacityDisabled:p,optionCheckColor:m,actionTextColor:h,optionColorPending:g,optionColorActive:_,loadingColor:v,loadingSize:y,optionColorActivePending:b,[O(`optionFontSize`,t)]:x,[O(`optionHeight`,t)]:C,[O(`optionPadding`,t)]:w}}=a.value;return{"--n-height":r,"--n-action-divider-color":c,"--n-action-text-color":h,"--n-bezier":n,"--n-border-radius":i,"--n-color":o,"--n-option-font-size":x,"--n-group-header-text-color":s,"--n-option-check-color":m,"--n-option-color-pending":g,"--n-option-color-active":_,"--n-option-color-active-pending":b,"--n-option-height":C,"--n-option-opacity-disabled":p,"--n-option-text-color":u,"--n-option-text-color-active":f,"--n-option-text-color-disabled":d,"--n-option-text-color-pressed":l,"--n-option-padding":w,"--n-option-padding-left":S(w,`left`),"--n-option-padding-right":S(w,`right`),"--n-loading-color":v,"--n-loading-size":y}}),{inlineThemeDisabled:J}=e,Y=J?te(`internal-select-menu`,A(()=>e.size[0]),ae,e):void 0,se={selfRef:o,next:re,prev:ie,getPendingTmNode:j};return Ae(o,e.onResize),Object.assign({mergedTheme:a,mergedClsPrefix:t,rtlEnabled:i,virtualListRef:c,scrollbarRef:d,itemSize:b,padding:x,flattenedNodes:f,empty:w,mergedRenderEmpty:T,virtualListContainer(){let{value:e}=c;return e?.listElRef},virtualListContent(){let{value:e}=c;return e?.itemsElRef},doScroll:D,handleFocusin:K,handleFocusout:q,handleKeyUp:R,handleKeyDown:z,handleMouseDown:B,handleVirtualListResize:k,handleVirtualListScroll:ee,cssVars:J?void 0:ae,themeClass:Y?.themeClass,onRender:Y?.onRender},se)},render(){let{$slots:e,virtualScroll:t,clsPrefix:n,mergedTheme:a,themeClass:o,onRender:s}=this;return s?.(),i(`div`,{ref:`selfRef`,tabindex:this.focusable?0:-1,class:[`${n}-base-select-menu`,`${n}-base-select-menu--${this.size}-size`,this.rtlEnabled&&`${n}-base-select-menu--rtl`,o,this.multiple&&`${n}-base-select-menu--multiple`],style:this.cssVars,onFocusin:this.handleFocusin,onFocusout:this.handleFocusout,onKeyup:this.handleKeyUp,onKeydown:this.handleKeyDown,onMousedown:this.handleMouseDown,onMouseenter:this.onMouseenter,onMouseleave:this.onMouseleave},B(e.header,e=>e&&i(`div`,{class:`${n}-base-select-menu__header`,"data-header":!0,key:`header`},e)),this.loading?i(`div`,{class:`${n}-base-select-menu__loading`},i(r,{clsPrefix:n,strokeWidth:20})):this.empty?i(`div`,{class:`${n}-base-select-menu__empty`,"data-empty":!0},re(e.empty,()=>[this.mergedRenderEmpty?.call(this)||i(fe,{theme:a.peers.Empty,themeOverrides:a.peerOverrides.Empty,size:this.size})])):i(T,Object.assign({ref:`scrollbarRef`,theme:a.peers.Scrollbar,themeOverrides:a.peerOverrides.Scrollbar,scrollable:this.scrollable,container:t?this.virtualListContainer:void 0,content:t?this.virtualListContent:void 0,onScroll:t?void 0:this.doScroll},this.scrollbarProps),{default:()=>t?i(ke,{ref:`virtualListRef`,class:`${n}-virtual-list`,items:this.flattenedNodes,itemSize:this.itemSize,showScrollbar:!1,paddingTop:this.padding.top,paddingBottom:this.padding.bottom,onResize:this.handleVirtualListResize,onScroll:this.handleVirtualListScroll,itemResizable:!0},{default:({item:e})=>e.isGroup?i(Fe,{key:e.key,clsPrefix:n,tmNode:e}):e.ignored?null:i(Le,{clsPrefix:n,key:e.key,tmNode:e})}):i(`div`,{class:`${n}-base-select-menu-option-wrapper`,style:{paddingTop:this.padding.top,paddingBottom:this.padding.bottom}},this.flattenedNodes.map(e=>e.isGroup?i(Fe,{key:e.key,clsPrefix:n,tmNode:e}):i(Le,{clsPrefix:n,key:e.key,tmNode:e})))}),B(e.action,e=>e&&[i(`div`,{class:`${n}-base-select-menu__action`,"data-action":!0,key:`action`},e),i(Pe,{onFocus:this.onTabOut,key:`focus-detector`})]))}}),Be=f([C(`base-selection`,`
 --n-padding-single: var(--n-padding-single-top) var(--n-padding-single-right) var(--n-padding-single-bottom) var(--n-padding-single-left);
 --n-padding-multiple: var(--n-padding-multiple-top) var(--n-padding-multiple-right) var(--n-padding-multiple-bottom) var(--n-padding-multiple-left);
 position: relative;
 z-index: auto;
 box-shadow: none;
 width: 100%;
 max-width: 100%;
 display: inline-block;
 vertical-align: bottom;
 border-radius: var(--n-border-radius);
 min-height: var(--n-height);
 line-height: 1.5;
 font-size: var(--n-font-size);
 `,[C(`base-loading`,`
 color: var(--n-loading-color);
 `),C(`base-selection-tags`,`min-height: var(--n-height);`),w(`border, state-border`,`
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 pointer-events: none;
 border: var(--n-border);
 border-radius: inherit;
 transition:
 box-shadow .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 `),w(`state-border`,`
 z-index: 1;
 border-color: #0000;
 `),C(`base-suffix`,`
 cursor: pointer;
 position: absolute;
 top: 50%;
 transform: translateY(-50%);
 right: 10px;
 `,[w(`arrow`,`
 font-size: var(--n-arrow-size);
 color: var(--n-arrow-color);
 transition: color .3s var(--n-bezier);
 `)]),C(`base-selection-overlay`,`
 display: flex;
 align-items: center;
 white-space: nowrap;
 pointer-events: none;
 position: absolute;
 top: 0;
 right: 0;
 bottom: 0;
 left: 0;
 padding: var(--n-padding-single);
 transition: color .3s var(--n-bezier);
 `,[w(`wrapper`,`
 flex-basis: 0;
 flex-grow: 1;
 overflow: hidden;
 text-overflow: ellipsis;
 `)]),C(`base-selection-placeholder`,`
 color: var(--n-placeholder-color);
 `,[w(`inner`,`
 max-width: 100%;
 overflow: hidden;
 `)]),C(`base-selection-tags`,`
 cursor: pointer;
 outline: none;
 box-sizing: border-box;
 position: relative;
 z-index: auto;
 display: flex;
 padding: var(--n-padding-multiple);
 flex-wrap: wrap;
 align-items: center;
 width: 100%;
 vertical-align: bottom;
 background-color: var(--n-color);
 border-radius: inherit;
 transition:
 color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier),
 background-color .3s var(--n-bezier);
 `),C(`base-selection-label`,`
 height: var(--n-height);
 display: inline-flex;
 width: 100%;
 vertical-align: bottom;
 cursor: pointer;
 outline: none;
 z-index: auto;
 box-sizing: border-box;
 position: relative;
 transition:
 color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier),
 background-color .3s var(--n-bezier);
 border-radius: inherit;
 background-color: var(--n-color);
 align-items: center;
 `,[C(`base-selection-input`,`
 font-size: inherit;
 line-height: inherit;
 outline: none;
 cursor: pointer;
 box-sizing: border-box;
 border:none;
 width: 100%;
 padding: var(--n-padding-single);
 background-color: #0000;
 color: var(--n-text-color);
 transition: color .3s var(--n-bezier);
 caret-color: var(--n-caret-color);
 `,[w(`content`,`
 text-overflow: ellipsis;
 overflow: hidden;
 white-space: nowrap; 
 `)]),w(`render-label`,`
 color: var(--n-text-color);
 `)]),e(`disabled`,[f(`&:hover`,[w(`state-border`,`
 box-shadow: var(--n-box-shadow-hover);
 border: var(--n-border-hover);
 `)]),x(`focus`,[w(`state-border`,`
 box-shadow: var(--n-box-shadow-focus);
 border: var(--n-border-focus);
 `)]),x(`active`,[w(`state-border`,`
 box-shadow: var(--n-box-shadow-active);
 border: var(--n-border-active);
 `),C(`base-selection-label`,`background-color: var(--n-color-active);`),C(`base-selection-tags`,`background-color: var(--n-color-active);`)])]),x(`disabled`,`cursor: not-allowed;`,[w(`arrow`,`
 color: var(--n-arrow-color-disabled);
 `),C(`base-selection-label`,`
 cursor: not-allowed;
 background-color: var(--n-color-disabled);
 `,[C(`base-selection-input`,`
 cursor: not-allowed;
 color: var(--n-text-color-disabled);
 `),w(`render-label`,`
 color: var(--n-text-color-disabled);
 `)]),C(`base-selection-tags`,`
 cursor: not-allowed;
 background-color: var(--n-color-disabled);
 `),C(`base-selection-placeholder`,`
 cursor: not-allowed;
 color: var(--n-placeholder-color-disabled);
 `)]),C(`base-selection-input-tag`,`
 height: calc(var(--n-height) - 6px);
 line-height: calc(var(--n-height) - 6px);
 outline: none;
 display: none;
 position: relative;
 margin-bottom: 3px;
 max-width: 100%;
 vertical-align: bottom;
 `,[w(`input`,`
 font-size: inherit;
 font-family: inherit;
 min-width: 1px;
 padding: 0;
 background-color: #0000;
 outline: none;
 border: none;
 max-width: 100%;
 overflow: hidden;
 width: 1em;
 line-height: inherit;
 cursor: pointer;
 color: var(--n-text-color);
 caret-color: var(--n-caret-color);
 `),w(`mirror`,`
 position: absolute;
 left: 0;
 top: 0;
 white-space: pre;
 visibility: hidden;
 user-select: none;
 -webkit-user-select: none;
 opacity: 0;
 `)]),[`warning`,`error`].map(t=>x(`${t}-status`,[w(`state-border`,`border: var(--n-border-${t});`),e(`disabled`,[f(`&:hover`,[w(`state-border`,`
 box-shadow: var(--n-box-shadow-hover-${t});
 border: var(--n-border-hover-${t});
 `)]),x(`active`,[w(`state-border`,`
 box-shadow: var(--n-box-shadow-active-${t});
 border: var(--n-border-active-${t});
 `),C(`base-selection-label`,`background-color: var(--n-color-active-${t});`),C(`base-selection-tags`,`background-color: var(--n-color-active-${t});`)]),x(`focus`,[w(`state-border`,`
 box-shadow: var(--n-box-shadow-focus-${t});
 border: var(--n-border-focus-${t});
 `)])])]))]),C(`base-selection-popover`,`
 margin-bottom: -3px;
 display: flex;
 flex-wrap: wrap;
 margin-right: -8px;
 `),C(`base-selection-tag-wrapper`,`
 max-width: 100%;
 display: inline-flex;
 padding: 0 7px 3px 0;
 `,[f(`&:last-child`,`padding-right: 0;`),C(`tag`,`
 font-size: 14px;
 max-width: 100%;
 `,[w(`content`,`
 line-height: 1.25;
 text-overflow: ellipsis;
 overflow: hidden;
 `)])])]),Ve=R({name:`InternalSelection`,props:Object.assign(Object.assign({},M.props),{clsPrefix:{type:String,required:!0},bordered:{type:Boolean,default:void 0},active:Boolean,pattern:{type:String,default:``},placeholder:String,selectedOption:{type:Object,default:null},selectedOptions:{type:Array,default:null},labelField:{type:String,default:`label`},valueField:{type:String,default:`value`},multiple:Boolean,filterable:Boolean,clearable:Boolean,disabled:Boolean,size:{type:String,default:`medium`},loading:Boolean,autofocus:Boolean,showArrow:{type:Boolean,default:!0},inputProps:Object,focused:Boolean,renderTag:Function,onKeydown:Function,onClick:Function,onBlur:Function,onFocus:Function,onDeleteOption:Function,maxTagCount:[String,Number],ellipsisTagPopoverProps:Object,onClear:Function,onPatternInput:Function,onPatternFocus:Function,onPatternBlur:Function,renderLabel:Function,status:String,inlineThemeDisabled:Boolean,ignoreComposition:{type:Boolean,default:!0},onResize:Function}),setup(e){let{mergedClsPrefixRef:t,mergedRtlRef:n}=ne(e),r=v(`InternalSelection`,n,t),i=I(null),a=I(null),o=I(null),c=I(null),l=I(null),d=I(null),f=I(null),m=I(null),h=I(null),g=I(null),_=I(!1),y=I(!1),b=I(!1),x=M(`InternalSelection`,`-internal-selection`,Be,me,e,L(e,`clsPrefix`)),C=A(()=>e.clearable&&!e.disabled&&(b.value||e.active)),w=A(()=>e.selectedOption?e.renderTag?e.renderTag({option:e.selectedOption,handleClose:()=>{}}):e.renderLabel?e.renderLabel(e.selectedOption,!0):$(e.selectedOption[e.labelField],e.selectedOption,!0):e.placeholder),T=A(()=>{let t=e.selectedOption;if(t)return t[e.labelField]}),E=A(()=>e.multiple?!!(Array.isArray(e.selectedOptions)&&e.selectedOptions.length):e.selectedOption!==null);function D(){var t;let{value:n}=i;if(n){let{value:r}=a;r&&(r.style.width=`${n.offsetWidth}px`,e.maxTagCount!==`responsive`&&((t=h.value)==null||t.sync({showAllItemsBeforeCalculate:!1})))}}function ee(){let{value:e}=g;e&&(e.style.display=`none`)}function k(){let{value:e}=g;e&&(e.style.display=`inline-block`)}s(L(e,`active`),e=>{e||ee()}),s(L(e,`pattern`),()=>{e.multiple&&N(D)});function j(t){let{onFocus:n}=e;n&&n(t)}function P(t){let{onBlur:n}=e;n&&n(t)}function F(t){let{onDeleteOption:n}=e;n&&n(t)}function R(t){let{onClear:n}=e;n&&n(t)}function z(t){let{onPatternInput:n}=e;n&&n(t)}function B(e){(!e.relatedTarget||!o.value?.contains(e.relatedTarget))&&j(e)}function re(e){o.value?.contains(e.relatedTarget)||P(e)}function V(e){R(e)}function ie(){b.value=!0}function H(){b.value=!1}function U(t){!e.active||!e.filterable||t.target!==a.value&&t.preventDefault()}function W(e){F(e)}let G=I(!1);function K(t){if(t.key===`Backspace`&&!G.value&&!e.pattern.length){let{selectedOptions:t}=e;t?.length&&W(t[t.length-1])}}let q=null;function ae(t){let{value:n}=i;n&&(n.textContent=t.target.value,D()),e.ignoreComposition&&G.value?q=t:z(t)}function oe(){G.value=!0}function J(){G.value=!1,e.ignoreComposition&&z(q),q=null}function Y(t){var n;y.value=!0,(n=e.onPatternFocus)==null||n.call(e,t)}function se(t){var n;y.value=!1,(n=e.onPatternBlur)==null||n.call(e,t)}function X(){var t,n;if(e.filterable)y.value=!1,(t=d.value)==null||t.blur(),(n=a.value)==null||n.blur();else if(e.multiple){let{value:e}=c;e?.blur()}else{let{value:e}=l;e?.blur()}}function Z(){var t,n,r;e.filterable?(y.value=!1,(t=d.value)==null||t.focus()):e.multiple?(n=c.value)==null||n.focus():(r=l.value)==null||r.focus()}function ce(){let{value:e}=a;e&&(k(),e.focus())}function le(){let{value:e}=a;e&&e.blur()}function ue(e){let{value:t}=f;t&&t.setTextContent(`+${e}`)}function de(){let{value:e}=m;return e}function Q(){return a.value}let fe=null;function pe(){fe!==null&&window.clearTimeout(fe)}function he(){e.active||(pe(),fe=window.setTimeout(()=>{E.value&&(_.value=!0)},100))}function ge(){pe()}function _e(e){e||(pe(),_.value=!1)}s(E,e=>{e||(_.value=!1)}),u(()=>{p(()=>{let t=d.value;t&&(e.disabled?t.removeAttribute(`tabindex`):t.tabIndex=y.value?-1:0)})}),Ae(o,e.onResize);let{inlineThemeDisabled:ve}=e,ye=A(()=>{let{size:t}=e,{common:{cubicBezierEaseInOut:n},self:{fontWeight:r,borderRadius:i,color:a,placeholderColor:o,textColor:s,paddingSingle:c,paddingMultiple:l,caretColor:u,colorDisabled:d,textColorDisabled:f,placeholderColorDisabled:p,colorActive:m,boxShadowFocus:h,boxShadowActive:g,boxShadowHover:_,border:v,borderFocus:y,borderHover:b,borderActive:C,arrowColor:w,arrowColorDisabled:T,loadingColor:E,colorActiveWarning:D,boxShadowFocusWarning:ee,boxShadowActiveWarning:k,boxShadowHoverWarning:te,borderWarning:A,borderFocusWarning:j,borderHoverWarning:M,borderActiveWarning:N,colorActiveError:P,boxShadowFocusError:F,boxShadowActiveError:ne,boxShadowHoverError:I,borderError:L,borderFocusError:R,borderHoverError:z,borderActiveError:B,clearColor:re,clearColorHover:V,clearColorPressed:ie,clearSize:H,arrowSize:U,[O(`height`,t)]:W,[O(`fontSize`,t)]:G}}=x.value,K=S(c),q=S(l);return{"--n-bezier":n,"--n-border":v,"--n-border-active":C,"--n-border-focus":y,"--n-border-hover":b,"--n-border-radius":i,"--n-box-shadow-active":g,"--n-box-shadow-focus":h,"--n-box-shadow-hover":_,"--n-caret-color":u,"--n-color":a,"--n-color-active":m,"--n-color-disabled":d,"--n-font-size":G,"--n-height":W,"--n-padding-single-top":K.top,"--n-padding-multiple-top":q.top,"--n-padding-single-right":K.right,"--n-padding-multiple-right":q.right,"--n-padding-single-left":K.left,"--n-padding-multiple-left":q.left,"--n-padding-single-bottom":K.bottom,"--n-padding-multiple-bottom":q.bottom,"--n-placeholder-color":o,"--n-placeholder-color-disabled":p,"--n-text-color":s,"--n-text-color-disabled":f,"--n-arrow-color":w,"--n-arrow-color-disabled":T,"--n-loading-color":E,"--n-color-active-warning":D,"--n-box-shadow-focus-warning":ee,"--n-box-shadow-active-warning":k,"--n-box-shadow-hover-warning":te,"--n-border-warning":A,"--n-border-focus-warning":j,"--n-border-hover-warning":M,"--n-border-active-warning":N,"--n-color-active-error":P,"--n-box-shadow-focus-error":F,"--n-box-shadow-active-error":ne,"--n-box-shadow-hover-error":I,"--n-border-error":L,"--n-border-focus-error":R,"--n-border-hover-error":z,"--n-border-active-error":B,"--n-clear-size":H,"--n-clear-color":re,"--n-clear-color-hover":V,"--n-clear-color-pressed":ie,"--n-arrow-size":U,"--n-font-weight":r}}),be=ve?te(`internal-selection`,A(()=>e.size[0]),ye,e):void 0;return{mergedTheme:x,mergedClearable:C,mergedClsPrefix:t,rtlEnabled:r,patternInputFocused:y,filterablePlaceholder:w,label:T,selected:E,showTagsPanel:_,isComposing:G,counterRef:f,counterWrapperRef:m,patternInputMirrorRef:i,patternInputRef:a,selfRef:o,multipleElRef:c,singleElRef:l,patternInputWrapperRef:d,overflowRef:h,inputTagElRef:g,handleMouseDown:U,handleFocusin:B,handleClear:V,handleMouseEnter:ie,handleMouseLeave:H,handleDeleteOption:W,handlePatternKeyDown:K,handlePatternInputInput:ae,handlePatternInputBlur:se,handlePatternInputFocus:Y,handleMouseEnterCounter:he,handleMouseLeaveCounter:ge,handleFocusout:re,handleCompositionEnd:J,handleCompositionStart:oe,onPopoverUpdateShow:_e,focus:Z,focusInput:ce,blur:X,blurInput:le,updateCounter:ue,getCounter:de,getTail:Q,renderLabel:e.renderLabel,cssVars:ve?void 0:ye,themeClass:be?.themeClass,onRender:be?.onRender}},render(){let{status:e,multiple:t,size:n,disabled:r,filterable:a,maxTagCount:o,bordered:s,clsPrefix:c,ellipsisTagPopoverProps:l,onRender:u,renderTag:d,renderLabel:f}=this;u?.();let p=o===`responsive`,m=typeof o==`number`,h=p||m,g=i(j,null,{default:()=>i(Q,{clsPrefix:c,loading:this.loading,showArrow:this.showArrow,showClear:this.mergedClearable&&this.selected,onClear:this.handleClear},{default:()=>{var e;return(e=this.$slots).arrow?.call(e)}})}),_;if(t){let{labelField:e}=this,t=t=>i(`div`,{class:`${c}-base-selection-tag-wrapper`,key:t.value},d?d({option:t,handleClose:()=>{this.handleDeleteOption(t)}}):i(ge,{size:n,closable:!t.disabled,disabled:r,onClose:()=>{this.handleDeleteOption(t)},internalCloseIsButtonTag:!1,internalCloseFocusable:!1},{default:()=>f?f(t,!0):$(t[e],t,!0)})),s=()=>(m?this.selectedOptions.slice(0,o):this.selectedOptions).map(t),u=a?i(`div`,{class:`${c}-base-selection-input-tag`,ref:`inputTagElRef`,key:`__input-tag__`},i(`input`,Object.assign({},this.inputProps,{ref:`patternInputRef`,tabindex:-1,disabled:r,value:this.pattern,autofocus:this.autofocus,class:`${c}-base-selection-input-tag__input`,onBlur:this.handlePatternInputBlur,onFocus:this.handlePatternInputFocus,onKeydown:this.handlePatternKeyDown,onInput:this.handlePatternInputInput,onCompositionstart:this.handleCompositionStart,onCompositionend:this.handleCompositionEnd})),i(`span`,{ref:`patternInputMirrorRef`,class:`${c}-base-selection-input-tag__mirror`},this.pattern)):null,v=p?()=>i(`div`,{class:`${c}-base-selection-tag-wrapper`,ref:`counterWrapperRef`},i(ge,{size:n,ref:`counterRef`,onMouseenter:this.handleMouseEnterCounter,onMouseleave:this.handleMouseLeaveCounter,disabled:r})):void 0,y;if(m){let e=this.selectedOptions.length-o;e>0&&(y=i(`div`,{class:`${c}-base-selection-tag-wrapper`,key:`__counter__`},i(ge,{size:n,ref:`counterRef`,onMouseenter:this.handleMouseEnterCounter,disabled:r},{default:()=>`+${e}`})))}let b=p?a?i(W,{ref:`overflowRef`,updateCounter:this.updateCounter,getCounter:this.getCounter,getTail:this.getTail,style:{width:`100%`,display:`flex`,overflow:`hidden`}},{default:s,counter:v,tail:()=>u}):i(W,{ref:`overflowRef`,updateCounter:this.updateCounter,getCounter:this.getCounter,style:{width:`100%`,display:`flex`,overflow:`hidden`}},{default:s,counter:v}):m&&y?s().concat(y):s(),x=h?()=>i(`div`,{class:`${c}-base-selection-popover`},p?s():this.selectedOptions.map(t)):void 0,S=h?Object.assign({show:this.showTagsPanel,trigger:`hover`,overlap:!0,placement:`top`,width:`trigger`,onUpdateShow:this.onPopoverUpdateShow,theme:this.mergedTheme.peers.Popover,themeOverrides:this.mergedTheme.peerOverrides.Popover},l):null,C=!this.selected&&(!this.active||!this.pattern&&!this.isComposing)?i(`div`,{class:`${c}-base-selection-placeholder ${c}-base-selection-overlay`},i(`div`,{class:`${c}-base-selection-placeholder__inner`},this.placeholder)):null,w=a?i(`div`,{ref:`patternInputWrapperRef`,class:`${c}-base-selection-tags`},b,p?null:u,g):i(`div`,{ref:`multipleElRef`,class:`${c}-base-selection-tags`,tabindex:r?void 0:0},b,g);_=i(k,null,h?i(Z,Object.assign({},S,{scrollable:!0,style:`max-height: calc(var(--v-target-height) * 6.6);`}),{trigger:()=>w,default:x}):w,C)}else if(a){let e=this.pattern||this.isComposing,t=this.active?!e:!this.selected,n=!this.active&&this.selected;_=i(`div`,{ref:`patternInputWrapperRef`,class:`${c}-base-selection-label`,title:this.patternInputFocused?void 0:je(this.label)},i(`input`,Object.assign({},this.inputProps,{ref:`patternInputRef`,class:`${c}-base-selection-input`,value:this.active?this.pattern:``,placeholder:``,readonly:r,disabled:r,tabindex:-1,autofocus:this.autofocus,onFocus:this.handlePatternInputFocus,onBlur:this.handlePatternInputBlur,onInput:this.handlePatternInputInput,onCompositionstart:this.handleCompositionStart,onCompositionend:this.handleCompositionEnd})),n?i(`div`,{class:`${c}-base-selection-label__render-label ${c}-base-selection-overlay`,key:`input`},i(`div`,{class:`${c}-base-selection-overlay__wrapper`},d?d({option:this.selectedOption,handleClose:()=>{}}):f?f(this.selectedOption,!0):$(this.label,this.selectedOption,!0))):null,t?i(`div`,{class:`${c}-base-selection-placeholder ${c}-base-selection-overlay`,key:`placeholder`},i(`div`,{class:`${c}-base-selection-overlay__wrapper`},this.filterablePlaceholder)):null,g)}else _=i(`div`,{ref:`singleElRef`,class:`${c}-base-selection-label`,tabindex:this.disabled?void 0:0},this.label===void 0?i(`div`,{class:`${c}-base-selection-placeholder ${c}-base-selection-overlay`,key:`placeholder`},i(`div`,{class:`${c}-base-selection-placeholder__inner`},this.placeholder)):i(`div`,{class:`${c}-base-selection-input`,title:je(this.label),key:`input`},i(`div`,{class:`${c}-base-selection-input__content`},d?d({option:this.selectedOption,handleClose:()=>{}}):f?f(this.selectedOption,!0):$(this.label,this.selectedOption,!0))),g);return i(`div`,{ref:`selfRef`,class:[`${c}-base-selection`,this.rtlEnabled&&`${c}-base-selection--rtl`,this.themeClass,e&&`${c}-base-selection--${e}-status`,{[`${c}-base-selection--active`]:this.active,[`${c}-base-selection--selected`]:this.selected||this.active&&this.pattern,[`${c}-base-selection--disabled`]:this.disabled,[`${c}-base-selection--multiple`]:this.multiple,[`${c}-base-selection--focus`]:this.focused}],style:this.cssVars,onClick:this.onClick,onMouseenter:this.handleMouseEnter,onMouseleave:this.handleMouseLeave,onKeydown:this.onKeydown,onFocusin:this.handleFocusin,onFocusout:this.handleFocusout,onMousedown:this.handleMouseDown},_,s?i(`div`,{class:`${c}-base-selection__border`}):null,s?i(`div`,{class:`${c}-base-selection__state-border`}):null)}});function He(e){return e.type===`group`}function Ue(e){return e.type===`ignored`}function We(e,t){try{return!!(1+t.toString().toLowerCase().indexOf(e.trim().toLowerCase()))}catch{return!1}}function Ge(e,t){return{getIsGroup:He,getIgnored:Ue,getKey(t){return He(t)?t.name||t.key||`key-required`:t[e]},getChildren(e){return e[t]}}}function Ke(e,t,n,r){if(!t)return e;function i(e){if(!Array.isArray(e))return[];let a=[];for(let o of e)if(He(o)){let e=i(o[r]);e.length&&a.push(Object.assign({},o,{[r]:e}))}else if(Ue(o))continue;else t(n,o)&&a.push(o);return a}return i(e)}function qe(e,t,n){let r=new Map;return e.forEach(e=>{He(e)?e[n].forEach(e=>{r.set(e[t],e)}):r.set(e[t],e)}),r}var Je=f([C(`select`,`
 z-index: auto;
 outline: none;
 width: 100%;
 position: relative;
 font-weight: var(--n-font-weight);
 `),C(`select-menu`,`
 margin: 4px 0;
 box-shadow: var(--n-menu-box-shadow);
 `,[_e({originalTransition:`background-color .3s var(--n-bezier), box-shadow .3s var(--n-bezier)`})])]),Ye=R({name:`Select`,props:Object.assign(Object.assign({},M.props),{to:Y.propTo,bordered:{type:Boolean,default:void 0},clearable:Boolean,clearCreatedOptionsOnClear:{type:Boolean,default:!0},clearFilterAfterSelect:{type:Boolean,default:!0},options:{type:Array,default:()=>[]},defaultValue:{type:[String,Number,Array],default:null},keyboard:{type:Boolean,default:!0},value:[String,Number,Array],placeholder:String,menuProps:Object,multiple:Boolean,size:String,menuSize:{type:String},filterable:Boolean,disabled:{type:Boolean,default:void 0},remote:Boolean,loading:Boolean,filter:Function,placement:{type:String,default:`bottom-start`},widthMode:{type:String,default:`trigger`},tag:Boolean,onCreate:Function,fallbackOption:{type:[Function,Boolean],default:void 0},show:{type:Boolean,default:void 0},showArrow:{type:Boolean,default:!0},maxTagCount:[Number,String],ellipsisTagPopoverProps:Object,consistentMenuWidth:{type:Boolean,default:!0},virtualScroll:{type:Boolean,default:!0},labelField:{type:String,default:`label`},valueField:{type:String,default:`value`},childrenField:{type:String,default:`children`},renderLabel:Function,renderOption:Function,renderTag:Function,"onUpdate:value":[Function,Array],inputProps:Object,nodeProps:Function,ignoreComposition:{type:Boolean,default:!0},showOnFocus:Boolean,onUpdateValue:[Function,Array],onBlur:[Function,Array],onClear:[Function,Array],onFocus:[Function,Array],onScroll:[Function,Array],onSearch:[Function,Array],onUpdateShow:[Function,Array],"onUpdate:show":[Function,Array],displayDirective:{type:String,default:`show`},resetMenuOnOptionsChange:{type:Boolean,default:!0},status:String,showCheckmark:{type:Boolean,default:!0},scrollbarProps:Object,onChange:[Function,Array],items:Array}),slots:Object,setup(e){let{mergedClsPrefixRef:t,mergedBorderedRef:n,namespaceRef:r,inlineThemeDisabled:i,mergedComponentPropsRef:a}=ne(e),o=M(`Select`,`-select`,Je,he,e,t),c=I(e.defaultValue),l=le(L(e,`value`),c),u=I(!1),d=I(``),f=ue(e,[`items`,`options`]),p=I([]),h=I([]),g=A(()=>h.value.concat(p.value).concat(f.value)),_=A(()=>{let{filter:t}=e;if(t)return t;let{labelField:n,valueField:r}=e;return(e,t)=>{if(!t)return!1;let i=t[n];if(typeof i==`string`)return We(e,i);let a=t[r];return typeof a==`string`?We(e,a):typeof a==`number`&&We(e,String(a))}}),v=A(()=>{if(e.remote)return f.value;{let{value:t}=g,{value:n}=d;return!n.length||!e.filterable?t:Ke(t,_.value,n,e.childrenField)}}),y=A(()=>{let{valueField:t,childrenField:n}=e,r=Ge(t,n);return G(v.value,r)}),x=A(()=>qe(g.value,e.valueField,e.childrenField)),S=I(!1),C=le(L(e,`show`),S),w=I(null),T=I(null),E=I(null),{localeRef:D}=de(`Select`),O=A(()=>e.placeholder??D.value.placeholder),k=[],j=I(new Map),N=A(()=>{let{fallbackOption:t}=e;if(t===void 0){let{labelField:t,valueField:n}=e;return e=>({[t]:String(e),[n]:e})}return t===!1?!1:e=>Object.assign(t(e),{value:e})});function P(t){let n=e.remote,{value:r}=j,{value:i}=x,{value:a}=N,o=[];return t.forEach(e=>{if(i.has(e))o.push(i.get(e));else if(n&&r.has(e))o.push(r.get(e));else if(a){let t=a(e);t&&o.push(t)}}),o}let F=A(()=>{if(e.multiple){let{value:e}=l;return Array.isArray(e)?P(e):[]}return null}),R=A(()=>{let{value:t}=l;return!e.multiple&&!Array.isArray(t)?t===null?null:P([t])[0]||null:null}),B=ee(e,{mergedSize:t=>{let{size:n}=e;if(n)return n;let{mergedSize:r}=t||{};return r?.value?r.value:a?.value?.Select?.size||`medium`}}),{mergedSizeRef:re,mergedDisabledRef:V,mergedStatusRef:ie}=B;function U(t,n){let{onChange:r,"onUpdate:value":i,onUpdateValue:a}=e,{nTriggerFormChange:o,nTriggerFormInput:s}=B;r&&z(r,t,n),a&&z(a,t,n),i&&z(i,t,n),c.value=t,o(),s()}function W(t){let{onBlur:n}=e,{nTriggerFormBlur:r}=B;n&&z(n,t),r()}function K(){let{onClear:t}=e;t&&z(t)}function q(t){let{onFocus:n,showOnFocus:r}=e,{nTriggerFormFocus:i}=B;n&&z(n,t),i(),r&&X()}function ae(t){let{onSearch:n}=e;n&&z(n,t)}function oe(t){let{onScroll:n}=e;n&&z(n,t)}function J(){var t;let{remote:n,multiple:r}=e;if(n){let{value:n}=j;if(r){let{valueField:r}=e;(t=F.value)==null||t.forEach(e=>{n.set(e[r],e)})}else{let t=R.value;t&&n.set(t[e.valueField],t)}}}function se(t){let{onUpdateShow:n,"onUpdate:show":r}=e;n&&z(n,t),r&&z(r,t),S.value=t}function X(){V.value||(se(!0),S.value=!0,e.filterable&&Me())}function Z(){se(!1)}function ce(){d.value=``,h.value=k}let Q=I(!1);function fe(){e.filterable&&(Q.value=!0)}function pe(){e.filterable&&(Q.value=!1,C.value||ce())}function me(){V.value||(C.value?e.filterable?Me():Z():X())}function ge(e){(E.value?.selfRef)?.contains(e.relatedTarget)||(u.value=!1,W(e),Z())}function _e(e){q(e),u.value=!0}function $(){u.value=!0}function ye(e){w.value?.$el.contains(e.relatedTarget)||(u.value=!1,W(e),Z())}function be(){var e;(e=w.value)==null||e.focus(),Z()}function xe(e){C.value&&(w.value?.$el.contains(m(e))||Z())}function Se(t){if(!Array.isArray(t))return[];if(N.value)return Array.from(t);{let{remote:n}=e,{value:r}=x;if(n){let{value:e}=j;return t.filter(t=>r.has(t)||e.has(t))}else return t.filter(e=>r.has(e))}}function Ce(e){we(e.rawNode)}function we(t){if(V.value)return;let{tag:n,remote:r,clearFilterAfterSelect:i,valueField:a}=e;if(n&&!r){let{value:e}=h,t=e[0]||null;if(t){let e=p.value;e.length?e.push(t):p.value=[t],h.value=k}}if(r&&j.value.set(t[a],t),e.multiple){let e=Se(l.value),o=e.findIndex(e=>e===t[a]);if(~o){if(e.splice(o,1),n&&!r){let e=Te(t[a]);~e&&(p.value.splice(e,1),i&&(d.value=``))}}else e.push(t[a]),i&&(d.value=``);U(e,P(e))}else{if(n&&!r){let e=Te(t[a]);~e?p.value=[p.value[e]]:p.value=k}je(),Z(),U(t[a],t)}}function Te(t){return p.value.findIndex(n=>n[e.valueField]===t)}function Ee(t){C.value||X();let{value:n}=t.target;d.value=n;let{tag:r,remote:i}=e;if(ae(n),r&&!i){if(!n){h.value=k;return}let{onCreate:t}=e,r=t?t(n):{[e.labelField]:n,[e.valueField]:n},{valueField:i,labelField:a}=e;f.value.some(e=>e[i]===r[i]||e[a]===r[a])||p.value.some(e=>e[i]===r[i]||e[a]===r[a])?h.value=k:h.value=[r]}}function De(t){t.stopPropagation();let{multiple:n,tag:r,remote:i,clearCreatedOptionsOnClear:a}=e;!n&&e.filterable&&Z(),r&&!i&&a&&(p.value=k),K(),n?U([],[]):U(null,null)}function Oe(e){!H(e,`action`)&&!H(e,`empty`)&&!H(e,`header`)&&e.preventDefault()}function ke(e){oe(e)}function Ae(t){var n,r,i;if(!e.keyboard){t.preventDefault();return}switch(t.key){case` `:if(e.filterable)break;t.preventDefault();case`Enter`:if(!w.value?.isComposing){if(C.value){let t=E.value?.getPendingTmNode();t?Ce(t):e.filterable||(Z(),je())}else if(X(),e.tag&&Q.value){let t=h.value[0];if(t){let n=t[e.valueField],{value:r}=l;e.multiple&&Array.isArray(r)&&r.includes(n)||we(t)}}}t.preventDefault();break;case`ArrowUp`:if(t.preventDefault(),e.loading)return;C.value&&((n=E.value)==null||n.prev());break;case`ArrowDown`:if(t.preventDefault(),e.loading)return;C.value?(r=E.value)==null||r.next():X();break;case`Escape`:C.value&&(ve(t),Z()),(i=w.value)==null||i.focus();break}}function je(){var e;(e=w.value)==null||e.focus()}function Me(){var e;(e=w.value)==null||e.focusInput()}function Ne(){var e;C.value&&((e=T.value)==null||e.syncPosition())}J(),s(L(e,`options`),J);let Pe={focus:()=>{var e;(e=w.value)==null||e.focus()},focusInput:()=>{var e;(e=w.value)==null||e.focusInput()},blur:()=>{var e;(e=w.value)==null||e.blur()},blurInput:()=>{var e;(e=w.value)==null||e.blurInput()}},Fe=A(()=>{let{self:{menuBoxShadow:e}}=o.value;return{"--n-menu-box-shadow":e}}),Ie=i?te(`select`,void 0,Fe,e):void 0;return Object.assign(Object.assign({},Pe),{mergedStatus:ie,mergedClsPrefix:t,mergedBordered:n,namespace:r,treeMate:y,isMounted:b(),triggerRef:w,menuRef:E,pattern:d,uncontrolledShow:S,mergedShow:C,adjustedTo:Y(e),uncontrolledValue:c,mergedValue:l,followerRef:T,localizedPlaceholder:O,selectedOption:R,selectedOptions:F,mergedSize:re,mergedDisabled:V,focused:u,activeWithoutMenuOpen:Q,inlineThemeDisabled:i,onTriggerInputFocus:fe,onTriggerInputBlur:pe,handleTriggerOrMenuResize:Ne,handleMenuFocus:$,handleMenuBlur:ye,handleMenuTabOut:be,handleTriggerClick:me,handleToggle:Ce,handleDeleteOption:we,handlePatternInput:Ee,handleClear:De,handleTriggerBlur:ge,handleTriggerFocus:_e,handleKeydown:Ae,handleMenuAfterLeave:ce,handleMenuClickOutside:xe,handleMenuScroll:ke,handleMenuKeydown:Ae,handleMenuMousedown:Oe,mergedTheme:o,cssVars:i?void 0:Fe,themeClass:Ie?.themeClass,onRender:Ie?.onRender})},render(){return i(`div`,{class:`${this.mergedClsPrefix}-select`},i(ae,null,{default:()=>[i(X,null,{default:()=>i(Ve,{ref:`triggerRef`,inlineThemeDisabled:this.inlineThemeDisabled,status:this.mergedStatus,inputProps:this.inputProps,clsPrefix:this.mergedClsPrefix,showArrow:this.showArrow,maxTagCount:this.maxTagCount,ellipsisTagPopoverProps:this.ellipsisTagPopoverProps,bordered:this.mergedBordered,active:this.activeWithoutMenuOpen||this.mergedShow,pattern:this.pattern,placeholder:this.localizedPlaceholder,selectedOption:this.selectedOption,selectedOptions:this.selectedOptions,multiple:this.multiple,renderTag:this.renderTag,renderLabel:this.renderLabel,filterable:this.filterable,clearable:this.clearable,disabled:this.mergedDisabled,size:this.mergedSize,theme:this.mergedTheme.peers.InternalSelection,labelField:this.labelField,valueField:this.valueField,themeOverrides:this.mergedTheme.peerOverrides.InternalSelection,loading:this.loading,focused:this.focused,onClick:this.handleTriggerClick,onDeleteOption:this.handleDeleteOption,onPatternInput:this.handlePatternInput,onClear:this.handleClear,onBlur:this.handleTriggerBlur,onFocus:this.handleTriggerFocus,onKeydown:this.handleKeydown,onPatternBlur:this.onTriggerInputBlur,onPatternFocus:this.onTriggerInputFocus,onResize:this.handleTriggerOrMenuResize,ignoreComposition:this.ignoreComposition},{arrow:()=>{var e;return[(e=this.$slots).arrow?.call(e)]}})}),i(J,{ref:`followerRef`,show:this.mergedShow,to:this.adjustedTo,teleportDisabled:this.adjustedTo===Y.tdkey,containerClass:this.namespace,width:this.consistentMenuWidth?`target`:void 0,minWidth:`target`,placement:this.placement},{default:()=>i(E,{name:`fade-in-scale-up-transition`,appear:this.isMounted,onAfterLeave:this.handleMenuAfterLeave},{default:()=>{var e;return this.mergedShow||this.displayDirective===`show`?((e=this.onRender)==null||e.call(this),d(i(ze,Object.assign({},this.menuProps,{ref:`menuRef`,onResize:this.handleTriggerOrMenuResize,inlineThemeDisabled:this.inlineThemeDisabled,virtualScroll:this.consistentMenuWidth&&this.virtualScroll,class:[`${this.mergedClsPrefix}-select-menu`,this.themeClass,this.menuProps?.class],clsPrefix:this.mergedClsPrefix,focusable:!0,labelField:this.labelField,valueField:this.valueField,autoPending:!0,nodeProps:this.nodeProps,theme:this.mergedTheme.peers.InternalSelectMenu,themeOverrides:this.mergedTheme.peerOverrides.InternalSelectMenu,treeMate:this.treeMate,multiple:this.multiple,size:this.menuSize,renderOption:this.renderOption,renderLabel:this.renderLabel,value:this.mergedValue,style:[this.menuProps?.style,this.cssVars],onToggle:this.handleToggle,onScroll:this.handleMenuScroll,onFocus:this.handleMenuFocus,onBlur:this.handleMenuBlur,onKeydown:this.handleMenuKeydown,onTabOut:this.handleMenuTabOut,onMousedown:this.handleMenuMousedown,show:this.mergedShow,showCheckmark:this.showCheckmark,resetMenuOnOptionsChange:this.resetMenuOnOptionsChange,scrollbarProps:this.scrollbarProps}),{empty:()=>{var e;return[(e=this.$slots).empty?.call(e)]},header:()=>{var e;return[(e=this.$slots).header?.call(e)]},action:()=>{var e;return[(e=this.$slots).action?.call(e)]}}),this.displayDirective===`show`?[[D,this.mergedShow],[K,this.handleMenuClickOutside,void 0,{capture:!0}]]:[[K,this.handleMenuClickOutside,void 0,{capture:!0}]])):null}})})]}))}});export{ke as a,Me as i,Ge as n,ze as r,Ye as t};