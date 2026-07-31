import{$t as e,An as t,At as n,C as r,En as i,Fn as a,Ft as o,Gn as s,Gt as c,Ht as l,In as u,Jt as d,Kt as f,Mn as p,On as m,Ot as h,P as g,Pn as _,Pt as v,Qt as y,Wn as b,Wt as x,Yt as S,Zt as C,_ as w,an as T,cn as E,dt as D,en as O,fn as k,ft as A,gn as j,gt as M,j as N,jn as P,k as F,kt as I,nr as L,or as R,pt as ee,qn as z,wn as B,wt as V,xt as H,yt as te,zn as U}from"./client-DzOxLNa2.js";import{t as ne}from"./next-frame-once-qdYFoq8G.js";import{i as W,n as G,r as K,t as q}from"./create-QAHl6yLj.js";import{l as J}from"./light-B3Jb2Ihz.js";import{a as Y,c as re,d as ie,i as X,l as ae,o as oe,s as Z,t as Q,u as se}from"./Popover-BF-SUuVR.js";import{a as ce,i as le}from"./text-DPkxR-eM.js";import{a as ue,r as de,s as fe,t as pe,u as me}from"./light-CbXzGyys.js";import{t as he}from"./use-locale-B2XeGYBF.js";import{n as ge}from"./Input-vgzMITwP.js";import{n as _e}from"./Icon-CBRp9WQI.js";import{k as $,w as ve}from"./index-CrTiSHWW.js";function ye(e){return e&-e}var be=class{constructor(e,t){this.l=e,this.min=t;let n=Array(e+1);for(let t=0;t<e+1;++t)n[t]=0;this.ft=n}add(e,t){if(t===0)return;let{l:n,ft:r}=this;for(e+=1;e<=n;)r[e]+=t,e+=ye(e)}get(e){return this.sum(e+1)-this.sum(e)}sum(e){if(e===void 0&&(e=this.l),e<=0)return 0;let{ft:t,min:n,l:r}=this;if(e>r)throw Error("[FinweckTree.sum]: `i` is larger than length.");let i=e*n;for(;e>0;)i+=t[e],e-=ye(e);return i}getBound(e){let t=0,n=this.l;for(;n>t;){let r=Math.floor((t+n)/2),i=this.sum(r);if(i>e){n=r;continue}else if(i<e){if(t===r)return this.sum(t+1)<=e?t+1:r;t=r}else return r}return t}},xe;function Se(){return typeof document>`u`?!1:(xe===void 0&&(xe=`matchMedia`in window&&window.matchMedia(`(pointer:coarse)`).matches),xe)}var Ce;function we(){return typeof document>`u`?1:(Ce===void 0&&(Ce=`chrome`in window?window.devicePixelRatio:1),Ce)}var Te=`VVirtualListXScroll`;function Ee({columnsRef:e,renderColRef:t,renderItemWithColsRef:n}){let r=L(0),i=L(0),a=j(()=>{let t=e.value;if(t.length===0)return null;let n=new be(t.length,0);return t.forEach((e,t)=>{n.add(t,e.width)}),n});return U(Te,{startIndexRef:o(()=>{let e=a.value;return e===null?0:Math.max(e.getBound(i.value)-1,0)}),endIndexRef:o(()=>{let t=a.value;return t===null?0:Math.min(t.getBound(i.value+r.value)+1,e.value.length-1)}),columnsRef:e,renderColRef:t,renderItemWithColsRef:n,getLeft:e=>{let t=a.value;return t===null?0:t.sum(e)}}),{listWidthRef:r,scrollLeftRef:i}}var De=B({name:`VirtualListRow`,props:{index:{type:Number,required:!0},item:{type:Object,required:!0}},setup(){let{startIndexRef:e,endIndexRef:t,columnsRef:n,getLeft:r,renderColRef:i,renderItemWithColsRef:a}=m(Te);return{startIndex:e,endIndex:t,columns:n,renderCol:i,renderItemWithCols:a,getLeft:r}},render(){let{startIndex:e,endIndex:t,columns:n,renderCol:r,renderItemWithCols:i,getLeft:a,item:o}=this;if(i!=null)return i({itemIndex:this.index,startColIndex:e,endColIndex:t,allColumns:n,item:o,getLeft:a});if(r!=null){let i=[];for(let s=e;s<=t;++s){let e=n[s];i.push(r({column:e,left:a(s),item:o}))}return i}return null}}),Oe=Y(`.v-vl`,{maxHeight:`inherit`,height:`100%`,overflow:`auto`,minWidth:`1px`},[Y(`&:not(.v-vl--show-scrollbar)`,{scrollbarWidth:`none`},[Y(`&::-webkit-scrollbar, &::-webkit-scrollbar-track-piece, &::-webkit-scrollbar-thumb`,{width:0,height:0,display:`none`})])]),ke=B({name:`VirtualList`,inheritAttrs:!1,props:{showScrollbar:{type:Boolean,default:!0},columns:{type:Array,default:()=>[]},renderCol:Function,renderItemWithCols:Function,items:{type:Array,default:()=>[]},itemSize:{type:Number,required:!0},itemResizable:Boolean,itemsStyle:[String,Object],visibleItemsTag:{type:[String,Object],default:`div`},visibleItemsProps:Object,ignoreItemResize:Boolean,onScroll:Function,onWheel:Function,onResize:Function,defaultScrollKey:[Number,String],defaultScrollIndex:Number,keyField:{type:String,default:`key`},paddingTop:{type:[Number,String],default:0},paddingBottom:{type:[Number,String],default:0}},setup(e){let t=n();Oe.mount({id:`vueuc/virtual-list`,head:!0,anchorMetaName:oe,ssr:t}),u(()=>{let{defaultScrollIndex:t,defaultScrollKey:n}=e;t==null?n!=null&&C({key:n}):C({index:t})});let r=!1,i=!1;p(()=>{if(r=!1,!i){i=!0;return}C({top:b.value,left:f.value})}),a(()=>{r=!0,i||=!0});let s=o(()=>{if(e.renderCol==null&&e.renderItemWithCols==null||e.columns.length===0)return;let t=0;return e.columns.forEach(e=>{t+=e.width}),t}),d=j(()=>{let t=new Map,{keyField:n}=e;return e.items.forEach((e,r)=>{t.set(e[n],r)}),t}),{scrollLeftRef:f,listWidthRef:m}=Ee({columnsRef:R(e,`columns`),renderColRef:R(e,`renderCol`),renderItemWithColsRef:R(e,`renderItemWithCols`)}),h=L(null),g=L(void 0),_=new Map,v=j(()=>{let{items:t,itemSize:n,keyField:r}=e,i=new be(t.length,n);return t.forEach((e,t)=>{let n=e[r],a=_.get(n);a!==void 0&&i.add(t,a)}),i}),y=L(0),b=L(0),x=o(()=>Math.max(v.value.getBound(b.value-l(e.paddingTop))-1,0)),S=j(()=>{let{value:t}=g;if(t===void 0)return[];let{items:n,itemSize:r}=e,i=x.value,a=Math.min(i+Math.ceil(t/r+1),n.length-1),o=[];for(let e=i;e<=a;++e)o.push(n[e]);return o}),C=(e,t)=>{if(typeof e==`number`){D(e,t,`auto`);return}let{left:n,top:r,index:i,key:a,position:o,behavior:s,debounce:c=!0}=e;if(n!==void 0||r!==void 0)D(n,r,s);else if(i!==void 0)E(i,s,c);else if(a!==void 0){let e=d.value.get(a);e!==void 0&&E(e,s,c)}else o===`bottom`?D(0,2**53-1,s):o===`top`&&D(0,0,s)},w,T=null;function E(t,n,r){let{value:i}=v,a=i.sum(t)+l(e.paddingTop);if(!r)h.value.scrollTo({left:0,top:a,behavior:n});else{w=t,T!==null&&window.clearTimeout(T),T=window.setTimeout(()=>{w=void 0,T=null},16);let{scrollTop:e,offsetHeight:r}=h.value;if(a>e){let o=i.get(t);a+o<=e+r||h.value.scrollTo({left:0,top:a+o-r,behavior:n})}else h.value.scrollTo({left:0,top:a,behavior:n})}}function D(e,t,n){h.value.scrollTo({left:e,top:t,behavior:n})}function O(t,n){if(r||e.ignoreItemResize||I(n.target))return;let{value:i}=v,a=d.value.get(t),o=i.get(a),s=n.borderBoxSize?.[0]?.blockSize??n.contentRect.height;if(s===o)return;s-e.itemSize===0?_.delete(t):_.set(t,s-e.itemSize);let c=s-o;if(c===0)return;i.add(a,c);let l=h.value;if(l!=null){if(w===void 0){let e=i.sum(a);l.scrollTop>e&&l.scrollBy(0,c)}else(a<w||a===w&&s+i.sum(a)>l.scrollTop+l.offsetHeight)&&l.scrollBy(0,c);F()}y.value++}let k=!Se(),A=!1;function M(t){var n;(n=e.onScroll)==null||n.call(e,t),(!k||!A)&&F()}function N(t){var n;if((n=e.onWheel)==null||n.call(e,t),k){let e=h.value;if(e!=null){if(t.deltaX===0&&(e.scrollTop===0&&t.deltaY<=0||e.scrollTop+e.offsetHeight>=e.scrollHeight&&t.deltaY>=0))return;t.preventDefault(),e.scrollTop+=t.deltaY/we(),e.scrollLeft+=t.deltaX/we(),F(),A=!0,ne(()=>{A=!1})}}}function P(t){if(r||I(t.target))return;if(e.renderCol==null&&e.renderItemWithCols==null){if(t.contentRect.height===g.value)return}else if(t.contentRect.height===g.value&&t.contentRect.width===m.value)return;g.value=t.contentRect.height,m.value=t.contentRect.width;let{onResize:n}=e;n!==void 0&&n(t)}function F(){let{value:e}=h;e!=null&&(b.value=e.scrollTop,f.value=e.scrollLeft)}function I(e){let t=e;for(;t!==null;){if(t.style.display===`none`)return!0;t=t.parentElement}return!1}return{listHeight:g,listStyle:{overflow:`auto`},keyToIndex:d,itemsStyle:j(()=>{let{itemResizable:t}=e,n=c(v.value.sum());return y.value,[e.itemsStyle,{boxSizing:`content-box`,width:c(s.value),height:t?``:n,minHeight:t?n:``,paddingTop:c(e.paddingTop),paddingBottom:c(e.paddingBottom)}]}),visibleItemsStyle:j(()=>(y.value,{transform:`translateY(${c(v.value.sum(x.value))})`})),viewportItems:S,listElRef:h,itemsElRef:L(null),scrollTo:C,handleListResize:P,handleListScroll:M,handleListWheel:N,handleItemResize:O}},render(){let{itemResizable:e,keyField:n,keyToIndex:r,visibleItemsTag:a}=this;return i(h,{onResize:this.handleListResize},{default:()=>{var o;return i(`div`,t(this.$attrs,{class:[`v-vl`,this.showScrollbar&&`v-vl--show-scrollbar`],onScroll:this.handleListScroll,onWheel:this.handleListWheel,ref:`listElRef`}),[this.items.length===0?(o=this.$slots).empty?.call(o):i(`div`,{ref:`itemsElRef`,class:`v-vl-items`,style:this.itemsStyle},[i(a,Object.assign({class:`v-vl-visible-items`,style:this.visibleItemsStyle},this.visibleItemsProps),{default:()=>{let{renderCol:t,renderItemWithCols:a}=this;return this.viewportItems.map(o=>{let s=o[n],c=r.get(s),l=t==null?void 0:i(De,{index:c,item:o}),u=a==null?void 0:i(De,{index:c,item:o}),d=this.$slots.default({item:o,renderedCols:l,renderedItemWithCols:u,index:c})[0];return e?i(h,{key:s,onResize:e=>this.handleItemResize(s,e)},{default:()=>d}):(d.key=s,d)})}})])])}})}});function Ae(e,t){t&&(u(()=>{let{value:n}=e;n&&I.registerHandler(n,t)}),b(e,(e,t)=>{t&&I.unregisterHandler(t)},{deep:!1}),_(()=>{let{value:t}=e;t&&I.unregisterHandler(t)}))}function je(e){switch(typeof e){case`string`:return e||void 0;case`number`:return String(e);default:return}}function Me(e){let t=e.filter(e=>e!==void 0);if(t.length!==0)return t.length===1?t[0]:t=>{e.forEach(e=>{e&&e(t)})}}var Ne=B({name:`Checkmark`,render(){return i(`svg`,{xmlns:`http://www.w3.org/2000/svg`,viewBox:`0 0 16 16`},i(`g`,{fill:`none`},i(`path`,{d:`M14.046 3.486a.75.75 0 0 1-.032 1.06l-7.93 7.474a.85.85 0 0 1-1.188-.022l-2.68-2.72a.75.75 0 1 1 1.068-1.053l2.234 2.267l7.468-7.038a.75.75 0 0 1 1.06.032z`,fill:`currentColor`})))}}),Pe=B({name:`Empty`,render(){return i(`svg`,{viewBox:`0 0 28 28`,fill:`none`,xmlns:`http://www.w3.org/2000/svg`},i(`path`,{d:`M26 7.5C26 11.0899 23.0899 14 19.5 14C15.9101 14 13 11.0899 13 7.5C13 3.91015 15.9101 1 19.5 1C23.0899 1 26 3.91015 26 7.5ZM16.8536 4.14645C16.6583 3.95118 16.3417 3.95118 16.1464 4.14645C15.9512 4.34171 15.9512 4.65829 16.1464 4.85355L18.7929 7.5L16.1464 10.1464C15.9512 10.3417 15.9512 10.6583 16.1464 10.8536C16.3417 11.0488 16.6583 11.0488 16.8536 10.8536L19.5 8.20711L22.1464 10.8536C22.3417 11.0488 22.6583 11.0488 22.8536 10.8536C23.0488 10.6583 23.0488 10.3417 22.8536 10.1464L20.2071 7.5L22.8536 4.85355C23.0488 4.65829 23.0488 4.34171 22.8536 4.14645C22.6583 3.95118 22.3417 3.95118 22.1464 4.14645L19.5 6.79289L16.8536 4.14645Z`,fill:`currentColor`}),i(`path`,{d:`M25 22.75V12.5991C24.5572 13.0765 24.053 13.4961 23.5 13.8454V16H17.5L17.3982 16.0068C17.0322 16.0565 16.75 16.3703 16.75 16.75C16.75 18.2688 15.5188 19.5 14 19.5C12.4812 19.5 11.25 18.2688 11.25 16.75L11.2432 16.6482C11.1935 16.2822 10.8797 16 10.5 16H4.5V7.25C4.5 6.2835 5.2835 5.5 6.25 5.5H12.2696C12.4146 4.97463 12.6153 4.47237 12.865 4H6.25C4.45507 4 3 5.45507 3 7.25V22.75C3 24.5449 4.45507 26 6.25 26H21.75C23.5449 26 25 24.5449 25 22.75ZM4.5 22.75V17.5H9.81597L9.85751 17.7041C10.2905 19.5919 11.9808 21 14 21L14.215 20.9947C16.2095 20.8953 17.842 19.4209 18.184 17.5H23.5V22.75C23.5 23.7165 22.7165 24.5 21.75 24.5H6.25C5.2835 24.5 4.5 23.7165 4.5 22.75Z`,fill:`currentColor`}))}}),Fe=B({props:{onFocus:Function,onBlur:Function},setup(e){return()=>i(`div`,{style:`width: 0; height: 0`,tabindex:0,onFocus:e.onFocus,onBlur:e.onBlur})}}),Ie=S(`empty`,`
 display: flex;
 flex-direction: column;
 align-items: center;
 font-size: var(--n-font-size);
`,[C(`icon`,`
 width: var(--n-icon-size);
 height: var(--n-icon-size);
 font-size: var(--n-icon-size);
 line-height: var(--n-icon-size);
 color: var(--n-icon-color);
 transition:
 color .3s var(--n-bezier);
 `,[d(`+`,[C(`description`,`
 margin-top: 8px;
 `)])]),C(`description`,`
 transition: color .3s var(--n-bezier);
 color: var(--n-text-color);
 `),C(`extra`,`
 text-align: center;
 transition: color .3s var(--n-bezier);
 margin-top: 12px;
 color: var(--n-extra-text-color);
 `)]),Le=B({name:`Empty`,props:Object.assign(Object.assign({},N.props),{description:String,showDescription:{type:Boolean,default:!0},showIcon:{type:Boolean,default:!0},size:{type:String,default:`medium`},renderIcon:Function}),slots:Object,setup(e){let{mergedClsPrefixRef:t,inlineThemeDisabled:n,mergedComponentPropsRef:r}=ee(e),a=N(`Empty`,`-empty`,Ie,fe,e,t),{localeRef:o}=he(`Empty`),s=j(()=>e.description??r?.value?.Empty?.description),c=j(()=>r?.value?.Empty?.renderIcon||(()=>i(Pe,null))),l=j(()=>{let{size:t}=e,{common:{cubicBezierEaseInOut:n},self:{[O(`iconSize`,t)]:r,[O(`fontSize`,t)]:i,textColor:o,iconColor:s,extraTextColor:c}}=a.value;return{"--n-icon-size":r,"--n-font-size":i,"--n-bezier":n,"--n-text-color":o,"--n-icon-color":s,"--n-extra-text-color":c}}),u=n?A(`empty`,j(()=>{let t=``,{size:n}=e;return t+=n[0],t}),l,e):void 0;return{mergedClsPrefix:t,mergedRenderIcon:c,localizedDescription:j(()=>s.value||o.value.description),cssVars:n?void 0:l,themeClass:u?.themeClass,onRender:u?.onRender}},render(){let{$slots:e,mergedClsPrefix:t,onRender:n}=this;return n?.(),i(`div`,{class:[`${t}-empty`,this.themeClass],style:this.cssVars},this.showIcon?i(`div`,{class:`${t}-empty__icon`},e.icon?e.icon():i(F,{clsPrefix:t},{default:this.mergedRenderIcon})):null,this.showDescription?i(`div`,{class:`${t}-empty__description`},e.default?e.default():this.localizedDescription):null,e.extra?i(`div`,{class:`${t}-empty__extra`},e.extra()):null)}}),Re=B({name:`NBaseSelectGroupHeader`,props:{clsPrefix:{type:String,required:!0},tmNode:{type:Object,required:!0}},setup(){let{renderLabelRef:e,renderOptionRef:t,labelFieldRef:n,nodePropsRef:r}=m(ie);return{labelField:n,nodeProps:r,renderLabel:e,renderOption:t}},render(){let{clsPrefix:e,renderLabel:t,renderOption:n,nodeProps:r,tmNode:{rawNode:a}}=this,o=r?.(a),s=t?t(a,!1):$(a[this.labelField],a,!1),c=i(`div`,Object.assign({},o,{class:[`${e}-base-select-group-header`,o?.class]}),s);return a.render?a.render({node:c,option:a}):n?n({node:c,option:a,selected:!1}):c}});function ze(e,t){return i(T,{name:`fade-in-scale-up-transition`},{default:()=>e?i(F,{clsPrefix:t,class:`${t}-base-select-option__check`},{default:()=>i(Ne)}):null})}var Be=B({name:`NBaseSelectOption`,props:{clsPrefix:{type:String,required:!0},tmNode:{type:Object,required:!0}},setup(e){let{valueRef:t,pendingTmNodeRef:n,multipleRef:r,valueSetRef:i,renderLabelRef:a,renderOptionRef:s,labelFieldRef:c,valueFieldRef:l,showCheckmarkRef:u,nodePropsRef:d,handleOptionClick:f,handleOptionMouseEnter:p}=m(ie),h=o(()=>{let{value:t}=n;return t?e.tmNode.key===t.key:!1});function g(t){let{tmNode:n}=e;n.disabled||f(t,n)}function _(t){let{tmNode:n}=e;n.disabled||p(t,n)}function v(t){let{tmNode:n}=e,{value:r}=h;n.disabled||r||p(t,n)}return{multiple:r,isGrouped:o(()=>{let{tmNode:t}=e,{parent:n}=t;return n&&n.rawNode.type===`group`}),showCheckmark:u,nodeProps:d,isPending:h,isSelected:o(()=>{let{value:n}=t,{value:a}=r;if(n===null)return!1;let o=e.tmNode.rawNode[l.value];if(a){let{value:e}=i;return e.has(o)}else return n===o}),labelField:c,renderLabel:a,renderOption:s,handleMouseMove:v,handleMouseEnter:_,handleClick:g}},render(){let{clsPrefix:e,tmNode:{rawNode:t},isSelected:n,isPending:r,isGrouped:a,showCheckmark:o,nodeProps:s,renderOption:c,renderLabel:l,handleClick:u,handleMouseEnter:d,handleMouseMove:f}=this,p=ze(n,e),m=l?[l(t,n),o&&p]:[$(t[this.labelField],t,n),o&&p],h=s?.(t),g=i(`div`,Object.assign({},h,{class:[`${e}-base-select-option`,t.class,h?.class,{[`${e}-base-select-option--disabled`]:t.disabled,[`${e}-base-select-option--selected`]:n,[`${e}-base-select-option--grouped`]:a,[`${e}-base-select-option--pending`]:r,[`${e}-base-select-option--show-checkmark`]:o}],style:[h?.style||``,t.style||``],onClick:Me([u,h?.onClick]),onMouseenter:Me([d,h?.onMouseenter]),onMousemove:Me([f,h?.onMousemove])}),i(`div`,{class:`${e}-base-select-option__content`},m));return t.render?t.render({node:g,option:t,selected:n}):c?c({node:g,option:t,selected:n}):g}}),Ve=S(`base-select-menu`,`
 line-height: 1.5;
 outline: none;
 z-index: 0;
 position: relative;
 border-radius: var(--n-border-radius);
 transition:
 background-color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier);
 background-color: var(--n-color);
`,[S(`scrollbar`,`
 max-height: var(--n-height);
 `),S(`virtual-list`,`
 max-height: var(--n-height);
 `),S(`base-select-option`,`
 min-height: var(--n-option-height);
 font-size: var(--n-option-font-size);
 display: flex;
 align-items: center;
 `,[C(`content`,`
 z-index: 1;
 white-space: nowrap;
 text-overflow: ellipsis;
 overflow: hidden;
 `)]),S(`base-select-group-header`,`
 min-height: var(--n-option-height);
 font-size: .93em;
 display: flex;
 align-items: center;
 `),S(`base-select-menu-option-wrapper`,`
 position: relative;
 width: 100%;
 `),C(`loading, empty`,`
 display: flex;
 padding: 12px 32px;
 flex: 1;
 justify-content: center;
 `),C(`loading`,`
 color: var(--n-loading-color);
 font-size: var(--n-loading-size);
 `),C(`header`,`
 padding: 8px var(--n-option-padding-left);
 font-size: var(--n-option-font-size);
 transition: 
 color .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 border-bottom: 1px solid var(--n-action-divider-color);
 color: var(--n-action-text-color);
 `),C(`action`,`
 padding: 8px var(--n-option-padding-left);
 font-size: var(--n-option-font-size);
 transition: 
 color .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 border-top: 1px solid var(--n-action-divider-color);
 color: var(--n-action-text-color);
 `),S(`base-select-group-header`,`
 position: relative;
 cursor: default;
 padding: var(--n-option-padding);
 color: var(--n-group-header-text-color);
 `),S(`base-select-option`,`
 cursor: pointer;
 position: relative;
 padding: var(--n-option-padding);
 transition:
 color .3s var(--n-bezier),
 opacity .3s var(--n-bezier);
 box-sizing: border-box;
 color: var(--n-option-text-color);
 opacity: 1;
 `,[y(`show-checkmark`,`
 padding-right: calc(var(--n-option-padding-right) + 20px);
 `),d(`&::before`,`
 content: "";
 position: absolute;
 left: 4px;
 right: 4px;
 top: 0;
 bottom: 0;
 border-radius: var(--n-border-radius);
 transition: background-color .3s var(--n-bezier);
 `),d(`&:active`,`
 color: var(--n-option-text-color-pressed);
 `),y(`grouped`,`
 padding-left: calc(var(--n-option-padding-left) * 1.5);
 `),y(`pending`,[d(`&::before`,`
 background-color: var(--n-option-color-pending);
 `)]),y(`selected`,`
 color: var(--n-option-text-color-active);
 `,[d(`&::before`,`
 background-color: var(--n-option-color-active);
 `),y(`pending`,[d(`&::before`,`
 background-color: var(--n-option-color-active-pending);
 `)])]),y(`disabled`,`
 cursor: not-allowed;
 `,[e(`selected`,`
 color: var(--n-option-text-color-disabled);
 `),y(`selected`,`
 opacity: var(--n-option-opacity-disabled);
 `)]),C(`check`,`
 font-size: 16px;
 position: absolute;
 right: calc(var(--n-option-padding-right) - 4px);
 top: calc(50% - 7px);
 color: var(--n-option-check-color);
 transition: color .3s var(--n-bezier);
 `,[ve({enterScale:`0.5`})])])]),He=B({name:`InternalSelectMenu`,props:Object.assign(Object.assign({},N.props),{clsPrefix:{type:String,required:!0},scrollable:{type:Boolean,default:!0},treeMate:{type:Object,required:!0},multiple:Boolean,size:{type:String,default:`medium`},value:{type:[String,Number,Array],default:null},autoPending:Boolean,virtualScroll:{type:Boolean,default:!0},show:{type:Boolean,default:!0},labelField:{type:String,default:`label`},valueField:{type:String,default:`value`},loading:Boolean,focusable:Boolean,renderLabel:Function,renderOption:Function,nodeProps:Function,showCheckmark:{type:Boolean,default:!0},onMousedown:Function,onScroll:Function,onFocus:Function,onBlur:Function,onKeyup:Function,onKeydown:Function,onTabOut:Function,onMouseenter:Function,onMouseleave:Function,onResize:Function,resetMenuOnOptionsChange:{type:Boolean,default:!0},inlineThemeDisabled:Boolean,scrollbarProps:Object,onToggle:Function}),setup(e){let{mergedClsPrefixRef:t,mergedRtlRef:n,mergedComponentPropsRef:r}=ee(e),i=g(`InternalSelectMenu`,n,t),a=N(`InternalSelectMenu`,`-internal-select-menu`,Ve,ue,e,R(e,`clsPrefix`)),o=L(null),s=L(null),c=L(null),d=j(()=>e.treeMate.getFlattenedNodes()),f=j(()=>G(d.value)),p=L(null);function m(){let{treeMate:t}=e,n=null,{value:r}=e;r===null?n=t.getFirstAvailableNode():(n=e.multiple?t.getNode((r||[])[(r||[]).length-1]):t.getNode(r),(!n||n.disabled)&&(n=t.getFirstAvailableNode())),K(n||null)}function h(){let{value:t}=p;t&&!e.treeMate.getNode(t.key)&&(p.value=null)}let v;b(()=>e.show,t=>{t?v=b(()=>e.treeMate,()=>{e.resetMenuOnOptionsChange?(e.autoPending?m():h(),P(q)):h()},{immediate:!0}):v?.()},{immediate:!0}),_(()=>{v?.()});let y=j(()=>l(a.value.self[O(`optionHeight`,e.size)])),S=j(()=>x(a.value.self[O(`padding`,e.size)])),C=j(()=>e.multiple&&Array.isArray(e.value)?new Set(e.value):new Set),w=j(()=>{let e=d.value;return e&&e.length===0}),T=j(()=>r?.value?.Select?.renderEmpty);function E(t){let{onToggle:n}=e;n&&n(t)}function D(t){let{onScroll:n}=e;n&&n(t)}function k(e){var t;(t=c.value)==null||t.sync(),D(e)}function M(){var e;(e=c.value)==null||e.sync()}function F(){let{value:e}=p;return e||null}function I(e,t){t.disabled||K(t,!1)}function z(e,t){t.disabled||E(t)}function B(t){var n;W(t,`action`)||(n=e.onKeyup)==null||n.call(e,t)}function V(t){var n;W(t,`action`)||(n=e.onKeydown)==null||n.call(e,t)}function H(t){var n;(n=e.onMousedown)==null||n.call(e,t),!e.focusable&&t.preventDefault()}function te(){let{value:e}=p;e&&K(e.getNext({loop:!0}),!0)}function ne(){let{value:e}=p;e&&K(e.getPrev({loop:!0}),!0)}function K(e,t=!1){p.value=e,t&&q()}function q(){var t,n;let r=p.value;if(!r)return;let i=f.value(r.key);i!==null&&(e.virtualScroll?(t=s.value)==null||t.scrollTo({index:i}):(n=c.value)==null||n.scrollTo({index:i,elSize:y.value}))}function J(t){var n;o.value?.contains(t.target)&&((n=e.onFocus)==null||n.call(e,t))}function Y(t){var n;o.value?.contains(t.relatedTarget)||(n=e.onBlur)==null||n.call(e,t)}U(ie,{handleOptionMouseEnter:I,handleOptionClick:z,valueSetRef:C,pendingTmNodeRef:p,nodePropsRef:R(e,`nodeProps`),showCheckmarkRef:R(e,`showCheckmark`),multipleRef:R(e,`multiple`),valueRef:R(e,`value`),renderLabelRef:R(e,`renderLabel`),renderOptionRef:R(e,`renderOption`),labelFieldRef:R(e,`labelField`),valueFieldRef:R(e,`valueField`)}),U(se,o),u(()=>{let{value:e}=c;e&&e.sync()});let re=j(()=>{let{size:t}=e,{common:{cubicBezierEaseInOut:n},self:{height:r,borderRadius:i,color:o,groupHeaderTextColor:s,actionDividerColor:c,optionTextColorPressed:l,optionTextColor:u,optionTextColorDisabled:d,optionTextColorActive:f,optionOpacityDisabled:p,optionCheckColor:m,actionTextColor:h,optionColorPending:g,optionColorActive:_,loadingColor:v,loadingSize:y,optionColorActivePending:b,[O(`optionFontSize`,t)]:S,[O(`optionHeight`,t)]:C,[O(`optionPadding`,t)]:w}}=a.value;return{"--n-height":r,"--n-action-divider-color":c,"--n-action-text-color":h,"--n-bezier":n,"--n-border-radius":i,"--n-color":o,"--n-option-font-size":S,"--n-group-header-text-color":s,"--n-option-check-color":m,"--n-option-color-pending":g,"--n-option-color-active":_,"--n-option-color-active-pending":b,"--n-option-height":C,"--n-option-opacity-disabled":p,"--n-option-text-color":u,"--n-option-text-color-active":f,"--n-option-text-color-disabled":d,"--n-option-text-color-pressed":l,"--n-option-padding":w,"--n-option-padding-left":x(w,`left`),"--n-option-padding-right":x(w,`right`),"--n-loading-color":v,"--n-loading-size":y}}),{inlineThemeDisabled:X}=e,ae=X?A(`internal-select-menu`,j(()=>e.size[0]),re,e):void 0,oe={selfRef:o,next:te,prev:ne,getPendingTmNode:F};return Ae(o,e.onResize),Object.assign({mergedTheme:a,mergedClsPrefix:t,rtlEnabled:i,virtualListRef:s,scrollbarRef:c,itemSize:y,padding:S,flattenedNodes:d,empty:w,mergedRenderEmpty:T,virtualListContainer(){let{value:e}=s;return e?.listElRef},virtualListContent(){let{value:e}=s;return e?.itemsElRef},doScroll:D,handleFocusin:J,handleFocusout:Y,handleKeyUp:B,handleKeyDown:V,handleMouseDown:H,handleVirtualListResize:M,handleVirtualListScroll:k,cssVars:X?void 0:re,themeClass:ae?.themeClass,onRender:ae?.onRender},oe)},render(){let{$slots:e,virtualScroll:t,clsPrefix:n,mergedTheme:a,themeClass:o,onRender:s}=this;return s?.(),i(`div`,{ref:`selfRef`,tabindex:this.focusable?0:-1,class:[`${n}-base-select-menu`,`${n}-base-select-menu--${this.size}-size`,this.rtlEnabled&&`${n}-base-select-menu--rtl`,o,this.multiple&&`${n}-base-select-menu--multiple`],style:this.cssVars,onFocusin:this.handleFocusin,onFocusout:this.handleFocusout,onKeyup:this.handleKeyUp,onKeydown:this.handleKeyDown,onMousedown:this.handleMouseDown,onMouseenter:this.onMouseenter,onMouseleave:this.onMouseleave},H(e.header,e=>e&&i(`div`,{class:`${n}-base-select-menu__header`,"data-header":!0,key:`header`},e)),this.loading?i(`div`,{class:`${n}-base-select-menu__loading`},i(r,{clsPrefix:n,strokeWidth:20})):this.empty?i(`div`,{class:`${n}-base-select-menu__empty`,"data-empty":!0},te(e.empty,()=>[this.mergedRenderEmpty?.call(this)||i(Le,{theme:a.peers.Empty,themeOverrides:a.peerOverrides.Empty,size:this.size})])):i(w,Object.assign({ref:`scrollbarRef`,theme:a.peers.Scrollbar,themeOverrides:a.peerOverrides.Scrollbar,scrollable:this.scrollable,container:t?this.virtualListContainer:void 0,content:t?this.virtualListContent:void 0,onScroll:t?void 0:this.doScroll},this.scrollbarProps),{default:()=>t?i(ke,{ref:`virtualListRef`,class:`${n}-virtual-list`,items:this.flattenedNodes,itemSize:this.itemSize,showScrollbar:!1,paddingTop:this.padding.top,paddingBottom:this.padding.bottom,onResize:this.handleVirtualListResize,onScroll:this.handleVirtualListScroll,itemResizable:!0},{default:({item:e})=>e.isGroup?i(Re,{key:e.key,clsPrefix:n,tmNode:e}):e.ignored?null:i(Be,{clsPrefix:n,key:e.key,tmNode:e})}):i(`div`,{class:`${n}-base-select-menu-option-wrapper`,style:{paddingTop:this.padding.top,paddingBottom:this.padding.bottom}},this.flattenedNodes.map(e=>e.isGroup?i(Re,{key:e.key,clsPrefix:n,tmNode:e}):i(Be,{clsPrefix:n,key:e.key,tmNode:e})))}),H(e.action,e=>e&&[i(`div`,{class:`${n}-base-select-menu__action`,"data-action":!0,key:`action`},e),i(Fe,{onFocus:this.onTabOut,key:`focus-detector`})]))}}),Ue=d([S(`base-selection`,`
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
 `,[S(`base-loading`,`
 color: var(--n-loading-color);
 `),S(`base-selection-tags`,`min-height: var(--n-height);`),C(`border, state-border`,`
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
 `),C(`state-border`,`
 z-index: 1;
 border-color: #0000;
 `),S(`base-suffix`,`
 cursor: pointer;
 position: absolute;
 top: 50%;
 transform: translateY(-50%);
 right: 10px;
 `,[C(`arrow`,`
 font-size: var(--n-arrow-size);
 color: var(--n-arrow-color);
 transition: color .3s var(--n-bezier);
 `)]),S(`base-selection-overlay`,`
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
 `,[C(`wrapper`,`
 flex-basis: 0;
 flex-grow: 1;
 overflow: hidden;
 text-overflow: ellipsis;
 `)]),S(`base-selection-placeholder`,`
 color: var(--n-placeholder-color);
 `,[C(`inner`,`
 max-width: 100%;
 overflow: hidden;
 `)]),S(`base-selection-tags`,`
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
 `),S(`base-selection-label`,`
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
 `,[S(`base-selection-input`,`
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
 `,[C(`content`,`
 text-overflow: ellipsis;
 overflow: hidden;
 white-space: nowrap; 
 `)]),C(`render-label`,`
 color: var(--n-text-color);
 `)]),e(`disabled`,[d(`&:hover`,[C(`state-border`,`
 box-shadow: var(--n-box-shadow-hover);
 border: var(--n-border-hover);
 `)]),y(`focus`,[C(`state-border`,`
 box-shadow: var(--n-box-shadow-focus);
 border: var(--n-border-focus);
 `)]),y(`active`,[C(`state-border`,`
 box-shadow: var(--n-box-shadow-active);
 border: var(--n-border-active);
 `),S(`base-selection-label`,`background-color: var(--n-color-active);`),S(`base-selection-tags`,`background-color: var(--n-color-active);`)])]),y(`disabled`,`cursor: not-allowed;`,[C(`arrow`,`
 color: var(--n-arrow-color-disabled);
 `),S(`base-selection-label`,`
 cursor: not-allowed;
 background-color: var(--n-color-disabled);
 `,[S(`base-selection-input`,`
 cursor: not-allowed;
 color: var(--n-text-color-disabled);
 `),C(`render-label`,`
 color: var(--n-text-color-disabled);
 `)]),S(`base-selection-tags`,`
 cursor: not-allowed;
 background-color: var(--n-color-disabled);
 `),S(`base-selection-placeholder`,`
 cursor: not-allowed;
 color: var(--n-placeholder-color-disabled);
 `)]),S(`base-selection-input-tag`,`
 height: calc(var(--n-height) - 6px);
 line-height: calc(var(--n-height) - 6px);
 outline: none;
 display: none;
 position: relative;
 margin-bottom: 3px;
 max-width: 100%;
 vertical-align: bottom;
 `,[C(`input`,`
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
 `),C(`mirror`,`
 position: absolute;
 left: 0;
 top: 0;
 white-space: pre;
 visibility: hidden;
 user-select: none;
 -webkit-user-select: none;
 opacity: 0;
 `)]),[`warning`,`error`].map(t=>y(`${t}-status`,[C(`state-border`,`border: var(--n-border-${t});`),e(`disabled`,[d(`&:hover`,[C(`state-border`,`
 box-shadow: var(--n-box-shadow-hover-${t});
 border: var(--n-border-hover-${t});
 `)]),y(`active`,[C(`state-border`,`
 box-shadow: var(--n-box-shadow-active-${t});
 border: var(--n-border-active-${t});
 `),S(`base-selection-label`,`background-color: var(--n-color-active-${t});`),S(`base-selection-tags`,`background-color: var(--n-color-active-${t});`)]),y(`focus`,[C(`state-border`,`
 box-shadow: var(--n-box-shadow-focus-${t});
 border: var(--n-border-focus-${t});
 `)])])]))]),S(`base-selection-popover`,`
 margin-bottom: -3px;
 display: flex;
 flex-wrap: wrap;
 margin-right: -8px;
 `),S(`base-selection-tag-wrapper`,`
 max-width: 100%;
 display: inline-flex;
 padding: 0 7px 3px 0;
 `,[d(`&:last-child`,`padding-right: 0;`),S(`tag`,`
 font-size: 14px;
 max-width: 100%;
 `,[C(`content`,`
 line-height: 1.25;
 text-overflow: ellipsis;
 overflow: hidden;
 `)])])]),We=B({name:`InternalSelection`,props:Object.assign(Object.assign({},N.props),{clsPrefix:{type:String,required:!0},bordered:{type:Boolean,default:void 0},active:Boolean,pattern:{type:String,default:``},placeholder:String,selectedOption:{type:Object,default:null},selectedOptions:{type:Array,default:null},labelField:{type:String,default:`label`},valueField:{type:String,default:`value`},multiple:Boolean,filterable:Boolean,clearable:Boolean,disabled:Boolean,size:{type:String,default:`medium`},loading:Boolean,autofocus:Boolean,showArrow:{type:Boolean,default:!0},inputProps:Object,focused:Boolean,renderTag:Function,onKeydown:Function,onClick:Function,onBlur:Function,onFocus:Function,onDeleteOption:Function,maxTagCount:[String,Number],ellipsisTagPopoverProps:Object,onClear:Function,onPatternInput:Function,onPatternFocus:Function,onPatternBlur:Function,renderLabel:Function,status:String,inlineThemeDisabled:Boolean,ignoreComposition:{type:Boolean,default:!0},onResize:Function}),setup(e){let{mergedClsPrefixRef:t,mergedRtlRef:n}=ee(e),r=g(`InternalSelection`,n,t),i=L(null),a=L(null),o=L(null),c=L(null),l=L(null),d=L(null),f=L(null),p=L(null),m=L(null),h=L(null),_=L(!1),v=L(!1),y=L(!1),S=N(`InternalSelection`,`-internal-selection`,Ue,de,e,R(e,`clsPrefix`)),C=j(()=>e.clearable&&!e.disabled&&(y.value||e.active)),w=j(()=>e.selectedOption?e.renderTag?e.renderTag({option:e.selectedOption,handleClose:()=>{}}):e.renderLabel?e.renderLabel(e.selectedOption,!0):$(e.selectedOption[e.labelField],e.selectedOption,!0):e.placeholder),T=j(()=>{let t=e.selectedOption;if(t)return t[e.labelField]}),E=j(()=>e.multiple?!!(Array.isArray(e.selectedOptions)&&e.selectedOptions.length):e.selectedOption!==null);function D(){var t;let{value:n}=i;if(n){let{value:r}=a;r&&(r.style.width=`${n.offsetWidth}px`,e.maxTagCount!==`responsive`&&((t=m.value)==null||t.sync({showAllItemsBeforeCalculate:!1})))}}function k(){let{value:e}=h;e&&(e.style.display=`none`)}function M(){let{value:e}=h;e&&(e.style.display=`inline-block`)}b(R(e,`active`),e=>{e||k()}),b(R(e,`pattern`),()=>{e.multiple&&P(D)});function F(t){let{onFocus:n}=e;n&&n(t)}function I(t){let{onBlur:n}=e;n&&n(t)}function z(t){let{onDeleteOption:n}=e;n&&n(t)}function B(t){let{onClear:n}=e;n&&n(t)}function V(t){let{onPatternInput:n}=e;n&&n(t)}function H(e){(!e.relatedTarget||!o.value?.contains(e.relatedTarget))&&F(e)}function te(e){o.value?.contains(e.relatedTarget)||I(e)}function U(e){B(e)}function ne(){y.value=!0}function W(){y.value=!1}function G(t){!e.active||!e.filterable||t.target!==a.value&&t.preventDefault()}function K(e){z(e)}let q=L(!1);function J(t){if(t.key===`Backspace`&&!q.value&&!e.pattern.length){let{selectedOptions:t}=e;t?.length&&K(t[t.length-1])}}let Y=null;function re(t){let{value:n}=i;n&&(n.textContent=t.target.value,D()),e.ignoreComposition&&q.value?Y=t:V(t)}function ie(){q.value=!0}function X(){q.value=!1,e.ignoreComposition&&V(Y),Y=null}function ae(t){var n;v.value=!0,(n=e.onPatternFocus)==null||n.call(e,t)}function oe(t){var n;v.value=!1,(n=e.onPatternBlur)==null||n.call(e,t)}function Z(){var t,n;if(e.filterable)v.value=!1,(t=d.value)==null||t.blur(),(n=a.value)==null||n.blur();else if(e.multiple){let{value:e}=c;e?.blur()}else{let{value:e}=l;e?.blur()}}function Q(){var t,n,r;e.filterable?(v.value=!1,(t=d.value)==null||t.focus()):e.multiple?(n=c.value)==null||n.focus():(r=l.value)==null||r.focus()}function se(){let{value:e}=a;e&&(M(),e.focus())}function ce(){let{value:e}=a;e&&e.blur()}function le(e){let{value:t}=f;t&&t.setTextContent(`+${e}`)}function ue(){let{value:e}=p;return e}function fe(){return a.value}let pe=null;function me(){pe!==null&&window.clearTimeout(pe)}function he(){e.active||(me(),pe=window.setTimeout(()=>{E.value&&(_.value=!0)},100))}function ge(){me()}function _e(e){e||(me(),_.value=!1)}b(E,e=>{e||(_.value=!1)}),u(()=>{s(()=>{let t=d.value;t&&(e.disabled?t.removeAttribute(`tabindex`):t.tabIndex=v.value?-1:0)})}),Ae(o,e.onResize);let{inlineThemeDisabled:ve}=e,ye=j(()=>{let{size:t}=e,{common:{cubicBezierEaseInOut:n},self:{fontWeight:r,borderRadius:i,color:a,placeholderColor:o,textColor:s,paddingSingle:c,paddingMultiple:l,caretColor:u,colorDisabled:d,textColorDisabled:f,placeholderColorDisabled:p,colorActive:m,boxShadowFocus:h,boxShadowActive:g,boxShadowHover:_,border:v,borderFocus:y,borderHover:b,borderActive:C,arrowColor:w,arrowColorDisabled:T,loadingColor:E,colorActiveWarning:D,boxShadowFocusWarning:k,boxShadowActiveWarning:A,boxShadowHoverWarning:j,borderWarning:M,borderFocusWarning:N,borderHoverWarning:P,borderActiveWarning:F,colorActiveError:I,boxShadowFocusError:L,boxShadowActiveError:R,boxShadowHoverError:ee,borderError:z,borderFocusError:B,borderHoverError:V,borderActiveError:H,clearColor:te,clearColorHover:U,clearColorPressed:ne,clearSize:W,arrowSize:G,[O(`height`,t)]:K,[O(`fontSize`,t)]:q}}=S.value,J=x(c),Y=x(l);return{"--n-bezier":n,"--n-border":v,"--n-border-active":C,"--n-border-focus":y,"--n-border-hover":b,"--n-border-radius":i,"--n-box-shadow-active":g,"--n-box-shadow-focus":h,"--n-box-shadow-hover":_,"--n-caret-color":u,"--n-color":a,"--n-color-active":m,"--n-color-disabled":d,"--n-font-size":q,"--n-height":K,"--n-padding-single-top":J.top,"--n-padding-multiple-top":Y.top,"--n-padding-single-right":J.right,"--n-padding-multiple-right":Y.right,"--n-padding-single-left":J.left,"--n-padding-multiple-left":Y.left,"--n-padding-single-bottom":J.bottom,"--n-padding-multiple-bottom":Y.bottom,"--n-placeholder-color":o,"--n-placeholder-color-disabled":p,"--n-text-color":s,"--n-text-color-disabled":f,"--n-arrow-color":w,"--n-arrow-color-disabled":T,"--n-loading-color":E,"--n-color-active-warning":D,"--n-box-shadow-focus-warning":k,"--n-box-shadow-active-warning":A,"--n-box-shadow-hover-warning":j,"--n-border-warning":M,"--n-border-focus-warning":N,"--n-border-hover-warning":P,"--n-border-active-warning":F,"--n-color-active-error":I,"--n-box-shadow-focus-error":L,"--n-box-shadow-active-error":R,"--n-box-shadow-hover-error":ee,"--n-border-error":z,"--n-border-focus-error":B,"--n-border-hover-error":V,"--n-border-active-error":H,"--n-clear-size":W,"--n-clear-color":te,"--n-clear-color-hover":U,"--n-clear-color-pressed":ne,"--n-arrow-size":G,"--n-font-weight":r}}),be=ve?A(`internal-selection`,j(()=>e.size[0]),ye,e):void 0;return{mergedTheme:S,mergedClearable:C,mergedClsPrefix:t,rtlEnabled:r,patternInputFocused:v,filterablePlaceholder:w,label:T,selected:E,showTagsPanel:_,isComposing:q,counterRef:f,counterWrapperRef:p,patternInputMirrorRef:i,patternInputRef:a,selfRef:o,multipleElRef:c,singleElRef:l,patternInputWrapperRef:d,overflowRef:m,inputTagElRef:h,handleMouseDown:G,handleFocusin:H,handleClear:U,handleMouseEnter:ne,handleMouseLeave:W,handleDeleteOption:K,handlePatternKeyDown:J,handlePatternInputInput:re,handlePatternInputBlur:oe,handlePatternInputFocus:ae,handleMouseEnterCounter:he,handleMouseLeaveCounter:ge,handleFocusout:te,handleCompositionEnd:X,handleCompositionStart:ie,onPopoverUpdateShow:_e,focus:Q,focusInput:se,blur:Z,blurInput:ce,updateCounter:le,getCounter:ue,getTail:fe,renderLabel:e.renderLabel,cssVars:ve?void 0:ye,themeClass:be?.themeClass,onRender:be?.onRender}},render(){let{status:e,multiple:t,size:n,disabled:r,filterable:a,maxTagCount:o,bordered:s,clsPrefix:c,ellipsisTagPopoverProps:l,onRender:u,renderTag:d,renderLabel:f}=this;u?.();let p=o===`responsive`,m=typeof o==`number`,h=p||m,g=i(M,null,{default:()=>i(ge,{clsPrefix:c,loading:this.loading,showArrow:this.showArrow,showClear:this.mergedClearable&&this.selected,onClear:this.handleClear},{default:()=>{var e;return(e=this.$slots).arrow?.call(e)}})}),_;if(t){let{labelField:e}=this,t=t=>i(`div`,{class:`${c}-base-selection-tag-wrapper`,key:t.value},d?d({option:t,handleClose:()=>{this.handleDeleteOption(t)}}):i(_e,{size:n,closable:!t.disabled,disabled:r,onClose:()=>{this.handleDeleteOption(t)},internalCloseIsButtonTag:!1,internalCloseFocusable:!1},{default:()=>f?f(t,!0):$(t[e],t,!0)})),s=()=>(m?this.selectedOptions.slice(0,o):this.selectedOptions).map(t),u=a?i(`div`,{class:`${c}-base-selection-input-tag`,ref:`inputTagElRef`,key:`__input-tag__`},i(`input`,Object.assign({},this.inputProps,{ref:`patternInputRef`,tabindex:-1,disabled:r,value:this.pattern,autofocus:this.autofocus,class:`${c}-base-selection-input-tag__input`,onBlur:this.handlePatternInputBlur,onFocus:this.handlePatternInputFocus,onKeydown:this.handlePatternKeyDown,onInput:this.handlePatternInputInput,onCompositionstart:this.handleCompositionStart,onCompositionend:this.handleCompositionEnd})),i(`span`,{ref:`patternInputMirrorRef`,class:`${c}-base-selection-input-tag__mirror`},this.pattern)):null,v=p?()=>i(`div`,{class:`${c}-base-selection-tag-wrapper`,ref:`counterWrapperRef`},i(_e,{size:n,ref:`counterRef`,onMouseenter:this.handleMouseEnterCounter,onMouseleave:this.handleMouseLeaveCounter,disabled:r})):void 0,y;if(m){let e=this.selectedOptions.length-o;e>0&&(y=i(`div`,{class:`${c}-base-selection-tag-wrapper`,key:`__counter__`},i(_e,{size:n,ref:`counterRef`,onMouseenter:this.handleMouseEnterCounter,disabled:r},{default:()=>`+${e}`})))}let b=p?a?i(K,{ref:`overflowRef`,updateCounter:this.updateCounter,getCounter:this.getCounter,getTail:this.getTail,style:{width:`100%`,display:`flex`,overflow:`hidden`}},{default:s,counter:v,tail:()=>u}):i(K,{ref:`overflowRef`,updateCounter:this.updateCounter,getCounter:this.getCounter,style:{width:`100%`,display:`flex`,overflow:`hidden`}},{default:s,counter:v}):m&&y?s().concat(y):s(),x=h?()=>i(`div`,{class:`${c}-base-selection-popover`},p?s():this.selectedOptions.map(t)):void 0,S=h?Object.assign({show:this.showTagsPanel,trigger:`hover`,overlap:!0,placement:`top`,width:`trigger`,onUpdateShow:this.onPopoverUpdateShow,theme:this.mergedTheme.peers.Popover,themeOverrides:this.mergedTheme.peerOverrides.Popover},l):null,C=!this.selected&&(!this.active||!this.pattern&&!this.isComposing)?i(`div`,{class:`${c}-base-selection-placeholder ${c}-base-selection-overlay`},i(`div`,{class:`${c}-base-selection-placeholder__inner`},this.placeholder)):null,w=a?i(`div`,{ref:`patternInputWrapperRef`,class:`${c}-base-selection-tags`},b,p?null:u,g):i(`div`,{ref:`multipleElRef`,class:`${c}-base-selection-tags`,tabindex:r?void 0:0},b,g);_=i(k,null,h?i(Q,Object.assign({},S,{scrollable:!0,style:`max-height: calc(var(--v-target-height) * 6.6);`}),{trigger:()=>w,default:x}):w,C)}else if(a){let e=this.pattern||this.isComposing,t=this.active?!e:!this.selected,n=!this.active&&this.selected;_=i(`div`,{ref:`patternInputWrapperRef`,class:`${c}-base-selection-label`,title:this.patternInputFocused?void 0:je(this.label)},i(`input`,Object.assign({},this.inputProps,{ref:`patternInputRef`,class:`${c}-base-selection-input`,value:this.active?this.pattern:``,placeholder:``,readonly:r,disabled:r,tabindex:-1,autofocus:this.autofocus,onFocus:this.handlePatternInputFocus,onBlur:this.handlePatternInputBlur,onInput:this.handlePatternInputInput,onCompositionstart:this.handleCompositionStart,onCompositionend:this.handleCompositionEnd})),n?i(`div`,{class:`${c}-base-selection-label__render-label ${c}-base-selection-overlay`,key:`input`},i(`div`,{class:`${c}-base-selection-overlay__wrapper`},d?d({option:this.selectedOption,handleClose:()=>{}}):f?f(this.selectedOption,!0):$(this.label,this.selectedOption,!0))):null,t?i(`div`,{class:`${c}-base-selection-placeholder ${c}-base-selection-overlay`,key:`placeholder`},i(`div`,{class:`${c}-base-selection-overlay__wrapper`},this.filterablePlaceholder)):null,g)}else _=i(`div`,{ref:`singleElRef`,class:`${c}-base-selection-label`,tabindex:this.disabled?void 0:0},this.label===void 0?i(`div`,{class:`${c}-base-selection-placeholder ${c}-base-selection-overlay`,key:`placeholder`},i(`div`,{class:`${c}-base-selection-placeholder__inner`},this.placeholder)):i(`div`,{class:`${c}-base-selection-input`,title:je(this.label),key:`input`},i(`div`,{class:`${c}-base-selection-input__content`},d?d({option:this.selectedOption,handleClose:()=>{}}):f?f(this.selectedOption,!0):$(this.label,this.selectedOption,!0))),g);return i(`div`,{ref:`selfRef`,class:[`${c}-base-selection`,this.rtlEnabled&&`${c}-base-selection--rtl`,this.themeClass,e&&`${c}-base-selection--${e}-status`,{[`${c}-base-selection--active`]:this.active,[`${c}-base-selection--selected`]:this.selected||this.active&&this.pattern,[`${c}-base-selection--disabled`]:this.disabled,[`${c}-base-selection--multiple`]:this.multiple,[`${c}-base-selection--focus`]:this.focused}],style:this.cssVars,onClick:this.onClick,onMouseenter:this.handleMouseEnter,onMouseleave:this.handleMouseLeave,onKeydown:this.onKeydown,onFocusin:this.handleFocusin,onFocusout:this.handleFocusout,onMousedown:this.handleMouseDown},_,s?i(`div`,{class:`${c}-base-selection__border`}):null,s?i(`div`,{class:`${c}-base-selection__state-border`}):null)}});function Ge(e){return e.type===`group`}function Ke(e){return e.type===`ignored`}function qe(e,t){try{return!!(1+t.toString().toLowerCase().indexOf(e.trim().toLowerCase()))}catch{return!1}}function Je(e,t){return{getIsGroup:Ge,getIgnored:Ke,getKey(t){return Ge(t)?t.name||t.key||`key-required`:t[e]},getChildren(e){return e[t]}}}function Ye(e,t,n,r){if(!t)return e;function i(e){if(!Array.isArray(e))return[];let a=[];for(let o of e)if(Ge(o)){let e=i(o[r]);e.length&&a.push(Object.assign({},o,{[r]:e}))}else if(Ke(o))continue;else t(n,o)&&a.push(o);return a}return i(e)}function Xe(e,t,n){let r=new Map;return e.forEach(e=>{Ge(e)?e[n].forEach(e=>{r.set(e[t],e)}):r.set(e[t],e)}),r}var Ze=d([S(`select`,`
 z-index: auto;
 outline: none;
 width: 100%;
 position: relative;
 font-weight: var(--n-font-weight);
 `),S(`select-menu`,`
 margin: 4px 0;
 box-shadow: var(--n-menu-box-shadow);
 `,[ve({originalTransition:`background-color .3s var(--n-bezier), box-shadow .3s var(--n-bezier)`})])]),Qe=B({name:`Select`,props:Object.assign(Object.assign({},N.props),{to:ae.propTo,bordered:{type:Boolean,default:void 0},clearable:Boolean,clearCreatedOptionsOnClear:{type:Boolean,default:!0},clearFilterAfterSelect:{type:Boolean,default:!0},options:{type:Array,default:()=>[]},defaultValue:{type:[String,Number,Array],default:null},keyboard:{type:Boolean,default:!0},value:[String,Number,Array],placeholder:String,menuProps:Object,multiple:Boolean,size:String,menuSize:{type:String},filterable:Boolean,disabled:{type:Boolean,default:void 0},remote:Boolean,loading:Boolean,filter:Function,placement:{type:String,default:`bottom-start`},widthMode:{type:String,default:`trigger`},tag:Boolean,onCreate:Function,fallbackOption:{type:[Function,Boolean],default:void 0},show:{type:Boolean,default:void 0},showArrow:{type:Boolean,default:!0},maxTagCount:[Number,String],ellipsisTagPopoverProps:Object,consistentMenuWidth:{type:Boolean,default:!0},virtualScroll:{type:Boolean,default:!0},labelField:{type:String,default:`label`},valueField:{type:String,default:`value`},childrenField:{type:String,default:`children`},renderLabel:Function,renderOption:Function,renderTag:Function,"onUpdate:value":[Function,Array],inputProps:Object,nodeProps:Function,ignoreComposition:{type:Boolean,default:!0},showOnFocus:Boolean,onUpdateValue:[Function,Array],onBlur:[Function,Array],onClear:[Function,Array],onFocus:[Function,Array],onScroll:[Function,Array],onSearch:[Function,Array],onUpdateShow:[Function,Array],"onUpdate:show":[Function,Array],displayDirective:{type:String,default:`show`},resetMenuOnOptionsChange:{type:Boolean,default:!0},status:String,showCheckmark:{type:Boolean,default:!0},scrollbarProps:Object,onChange:[Function,Array],items:Array}),slots:Object,setup(e){let{mergedClsPrefixRef:t,mergedBorderedRef:n,namespaceRef:r,inlineThemeDisabled:i,mergedComponentPropsRef:a}=ee(e),o=N(`Select`,`-select`,Ze,pe,e,t),s=L(e.defaultValue),c=ce(R(e,`value`),s),l=L(!1),u=L(``),d=le(e,[`items`,`options`]),p=L([]),m=L([]),h=j(()=>m.value.concat(p.value).concat(d.value)),g=j(()=>{let{filter:t}=e;if(t)return t;let{labelField:n,valueField:r}=e;return(e,t)=>{if(!t)return!1;let i=t[n];if(typeof i==`string`)return qe(e,i);let a=t[r];return typeof a==`string`?qe(e,a):typeof a==`number`&&qe(e,String(a))}}),_=j(()=>{if(e.remote)return d.value;{let{value:t}=h,{value:n}=u;return!n.length||!e.filterable?t:Ye(t,g.value,n,e.childrenField)}}),y=j(()=>{let{valueField:t,childrenField:n}=e,r=Je(t,n);return q(_.value,r)}),x=j(()=>Xe(h.value,e.valueField,e.childrenField)),S=L(!1),C=ce(R(e,`show`),S),w=L(null),T=L(null),E=L(null),{localeRef:O}=he(`Select`),k=j(()=>e.placeholder??O.value.placeholder),M=[],P=L(new Map),F=j(()=>{let{fallbackOption:t}=e;if(t===void 0){let{labelField:t,valueField:n}=e;return e=>({[t]:String(e),[n]:e})}return t===!1?!1:e=>Object.assign(t(e),{value:e})});function I(t){let n=e.remote,{value:r}=P,{value:i}=x,{value:a}=F,o=[];return t.forEach(e=>{if(i.has(e))o.push(i.get(e));else if(n&&r.has(e))o.push(r.get(e));else if(a){let t=a(e);t&&o.push(t)}}),o}let z=j(()=>{if(e.multiple){let{value:e}=c;return Array.isArray(e)?I(e):[]}return null}),B=j(()=>{let{value:t}=c;return!e.multiple&&!Array.isArray(t)?t===null?null:I([t])[0]||null:null}),H=D(e,{mergedSize:t=>{let{size:n}=e;if(n)return n;let{mergedSize:r}=t||{};return r?.value?r.value:a?.value?.Select?.size||`medium`}}),{mergedSizeRef:te,mergedDisabledRef:U,mergedStatusRef:ne}=H;function G(t,n){let{onChange:r,"onUpdate:value":i,onUpdateValue:a}=e,{nTriggerFormChange:o,nTriggerFormInput:c}=H;r&&V(r,t,n),a&&V(a,t,n),i&&V(i,t,n),s.value=t,o(),c()}function K(t){let{onBlur:n}=e,{nTriggerFormBlur:r}=H;n&&V(n,t),r()}function J(){let{onClear:t}=e;t&&V(t)}function Y(t){let{onFocus:n,showOnFocus:r}=e,{nTriggerFormFocus:i}=H;n&&V(n,t),i(),r&&Z()}function re(t){let{onSearch:n}=e;n&&V(n,t)}function ie(t){let{onScroll:n}=e;n&&V(n,t)}function X(){var t;let{remote:n,multiple:r}=e;if(n){let{value:n}=P;if(r){let{valueField:r}=e;(t=z.value)==null||t.forEach(e=>{n.set(e[r],e)})}else{let t=B.value;t&&n.set(t[e.valueField],t)}}}function oe(t){let{onUpdateShow:n,"onUpdate:show":r}=e;n&&V(n,t),r&&V(r,t),S.value=t}function Z(){U.value||(oe(!0),S.value=!0,e.filterable&&Me())}function Q(){oe(!1)}function se(){u.value=``,m.value=M}let ue=L(!1);function de(){e.filterable&&(ue.value=!0)}function fe(){e.filterable&&(ue.value=!1,C.value||se())}function ge(){U.value||(C.value?e.filterable?Me():Q():Z())}function _e(e){(E.value?.selfRef)?.contains(e.relatedTarget)||(l.value=!1,K(e),Q())}function $(e){Y(e),l.value=!0}function ve(){l.value=!0}function ye(e){w.value?.$el.contains(e.relatedTarget)||(l.value=!1,K(e),Q())}function be(){var e;(e=w.value)==null||e.focus(),Q()}function xe(e){C.value&&(w.value?.$el.contains(f(e))||Q())}function Se(t){if(!Array.isArray(t))return[];if(F.value)return Array.from(t);{let{remote:n}=e,{value:r}=x;if(n){let{value:e}=P;return t.filter(t=>r.has(t)||e.has(t))}else return t.filter(e=>r.has(e))}}function Ce(e){we(e.rawNode)}function we(t){if(U.value)return;let{tag:n,remote:r,clearFilterAfterSelect:i,valueField:a}=e;if(n&&!r){let{value:e}=m,t=e[0]||null;if(t){let e=p.value;e.length?e.push(t):p.value=[t],m.value=M}}if(r&&P.value.set(t[a],t),e.multiple){let e=Se(c.value),o=e.findIndex(e=>e===t[a]);if(~o){if(e.splice(o,1),n&&!r){let e=Te(t[a]);~e&&(p.value.splice(e,1),i&&(u.value=``))}}else e.push(t[a]),i&&(u.value=``);G(e,I(e))}else{if(n&&!r){let e=Te(t[a]);~e?p.value=[p.value[e]]:p.value=M}je(),Q(),G(t[a],t)}}function Te(t){return p.value.findIndex(n=>n[e.valueField]===t)}function Ee(t){C.value||Z();let{value:n}=t.target;u.value=n;let{tag:r,remote:i}=e;if(re(n),r&&!i){if(!n){m.value=M;return}let{onCreate:t}=e,r=t?t(n):{[e.labelField]:n,[e.valueField]:n},{valueField:i,labelField:a}=e;d.value.some(e=>e[i]===r[i]||e[a]===r[a])||p.value.some(e=>e[i]===r[i]||e[a]===r[a])?m.value=M:m.value=[r]}}function De(t){t.stopPropagation();let{multiple:n,tag:r,remote:i,clearCreatedOptionsOnClear:a}=e;!n&&e.filterable&&Q(),r&&!i&&a&&(p.value=M),J(),n?G([],[]):G(null,null)}function Oe(e){!W(e,`action`)&&!W(e,`empty`)&&!W(e,`header`)&&e.preventDefault()}function ke(e){ie(e)}function Ae(t){var n,r,i;if(!e.keyboard){t.preventDefault();return}switch(t.key){case` `:if(e.filterable)break;t.preventDefault();case`Enter`:if(!w.value?.isComposing){if(C.value){let t=E.value?.getPendingTmNode();t?Ce(t):e.filterable||(Q(),je())}else if(Z(),e.tag&&ue.value){let t=m.value[0];if(t){let n=t[e.valueField],{value:r}=c;e.multiple&&Array.isArray(r)&&r.includes(n)||we(t)}}}t.preventDefault();break;case`ArrowUp`:if(t.preventDefault(),e.loading)return;C.value&&((n=E.value)==null||n.prev());break;case`ArrowDown`:if(t.preventDefault(),e.loading)return;C.value?(r=E.value)==null||r.next():Z();break;case`Escape`:C.value&&(me(t),Q()),(i=w.value)==null||i.focus();break}}function je(){var e;(e=w.value)==null||e.focus()}function Me(){var e;(e=w.value)==null||e.focusInput()}function Ne(){var e;C.value&&((e=T.value)==null||e.syncPosition())}X(),b(R(e,`options`),X);let Pe={focus:()=>{var e;(e=w.value)==null||e.focus()},focusInput:()=>{var e;(e=w.value)==null||e.focusInput()},blur:()=>{var e;(e=w.value)==null||e.blur()},blurInput:()=>{var e;(e=w.value)==null||e.blurInput()}},Fe=j(()=>{let{self:{menuBoxShadow:e}}=o.value;return{"--n-menu-box-shadow":e}}),Ie=i?A(`select`,void 0,Fe,e):void 0;return Object.assign(Object.assign({},Pe),{mergedStatus:ne,mergedClsPrefix:t,mergedBordered:n,namespace:r,treeMate:y,isMounted:v(),triggerRef:w,menuRef:E,pattern:u,uncontrolledShow:S,mergedShow:C,adjustedTo:ae(e),uncontrolledValue:s,mergedValue:c,followerRef:T,localizedPlaceholder:k,selectedOption:B,selectedOptions:z,mergedSize:te,mergedDisabled:U,focused:l,activeWithoutMenuOpen:ue,inlineThemeDisabled:i,onTriggerInputFocus:de,onTriggerInputBlur:fe,handleTriggerOrMenuResize:Ne,handleMenuFocus:ve,handleMenuBlur:ye,handleMenuTabOut:be,handleTriggerClick:ge,handleToggle:Ce,handleDeleteOption:we,handlePatternInput:Ee,handleClear:De,handleTriggerBlur:_e,handleTriggerFocus:$,handleKeydown:Ae,handleMenuAfterLeave:se,handleMenuClickOutside:xe,handleMenuScroll:ke,handleMenuKeydown:Ae,handleMenuMousedown:Oe,mergedTheme:o,cssVars:i?void 0:Fe,themeClass:Ie?.themeClass,onRender:Ie?.onRender})},render(){return i(`div`,{class:`${this.mergedClsPrefix}-select`},i(re,null,{default:()=>[i(Z,null,{default:()=>i(We,{ref:`triggerRef`,inlineThemeDisabled:this.inlineThemeDisabled,status:this.mergedStatus,inputProps:this.inputProps,clsPrefix:this.mergedClsPrefix,showArrow:this.showArrow,maxTagCount:this.maxTagCount,ellipsisTagPopoverProps:this.ellipsisTagPopoverProps,bordered:this.mergedBordered,active:this.activeWithoutMenuOpen||this.mergedShow,pattern:this.pattern,placeholder:this.localizedPlaceholder,selectedOption:this.selectedOption,selectedOptions:this.selectedOptions,multiple:this.multiple,renderTag:this.renderTag,renderLabel:this.renderLabel,filterable:this.filterable,clearable:this.clearable,disabled:this.mergedDisabled,size:this.mergedSize,theme:this.mergedTheme.peers.InternalSelection,labelField:this.labelField,valueField:this.valueField,themeOverrides:this.mergedTheme.peerOverrides.InternalSelection,loading:this.loading,focused:this.focused,onClick:this.handleTriggerClick,onDeleteOption:this.handleDeleteOption,onPatternInput:this.handlePatternInput,onClear:this.handleClear,onBlur:this.handleTriggerBlur,onFocus:this.handleTriggerFocus,onKeydown:this.handleKeydown,onPatternBlur:this.onTriggerInputBlur,onPatternFocus:this.onTriggerInputFocus,onResize:this.handleTriggerOrMenuResize,ignoreComposition:this.ignoreComposition},{arrow:()=>{var e;return[(e=this.$slots).arrow?.call(e)]}})}),i(X,{ref:`followerRef`,show:this.mergedShow,to:this.adjustedTo,teleportDisabled:this.adjustedTo===ae.tdkey,containerClass:this.namespace,width:this.consistentMenuWidth?`target`:void 0,minWidth:`target`,placement:this.placement},{default:()=>i(T,{name:`fade-in-scale-up-transition`,appear:this.isMounted,onAfterLeave:this.handleMenuAfterLeave},{default:()=>{var e;return this.mergedShow||this.displayDirective===`show`?((e=this.onRender)==null||e.call(this),z(i(He,Object.assign({},this.menuProps,{ref:`menuRef`,onResize:this.handleTriggerOrMenuResize,inlineThemeDisabled:this.inlineThemeDisabled,virtualScroll:this.consistentMenuWidth&&this.virtualScroll,class:[`${this.mergedClsPrefix}-select-menu`,this.themeClass,this.menuProps?.class],clsPrefix:this.mergedClsPrefix,focusable:!0,labelField:this.labelField,valueField:this.valueField,autoPending:!0,nodeProps:this.nodeProps,theme:this.mergedTheme.peers.InternalSelectMenu,themeOverrides:this.mergedTheme.peerOverrides.InternalSelectMenu,treeMate:this.treeMate,multiple:this.multiple,size:this.menuSize,renderOption:this.renderOption,renderLabel:this.renderLabel,value:this.mergedValue,style:[this.menuProps?.style,this.cssVars],onToggle:this.handleToggle,onScroll:this.handleMenuScroll,onFocus:this.handleMenuFocus,onBlur:this.handleMenuBlur,onKeydown:this.handleMenuKeydown,onTabOut:this.handleMenuTabOut,onMousedown:this.handleMenuMousedown,show:this.mergedShow,showCheckmark:this.showCheckmark,resetMenuOnOptionsChange:this.resetMenuOnOptionsChange,scrollbarProps:this.scrollbarProps}),{empty:()=>{var e;return[(e=this.$slots).empty?.call(e)]},header:()=>{var e;return[(e=this.$slots).header?.call(e)]},action:()=>{var e;return[(e=this.$slots).action?.call(e)]}}),this.displayDirective===`show`?[[E,this.mergedShow],[J,this.handleMenuClickOutside,void 0,{capture:!0}]]:[[J,this.handleMenuClickOutside,void 0,{capture:!0}]])):null}})})]}))}});export{Me as a,Le as i,Je as n,ke as o,He as r,Qe as t};