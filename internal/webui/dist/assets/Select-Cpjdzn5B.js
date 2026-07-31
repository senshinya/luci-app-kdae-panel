import{$t as e,An as t,At as n,C as r,E as i,En as a,Fn as o,Ft as s,Gn as c,Gt as l,Ht as u,In as d,Jt as f,Kt as p,Lt as m,Mn as h,Nt as g,O as _,On as v,Ot as y,P as b,Pn as x,Pt as S,Qt as C,Wn as w,Wt as T,Yt as E,Zt as D,_ as ee,an as O,cn as k,dt as A,en as j,fn as M,ft as N,gn as P,gt as F,j as I,jn as te,k as L,kt as R,nn as z,nr as B,or as V,pt as H,qn as ne,tn as U,wn as W,wt as G,xt as K,yt as q,zn as J}from"./client-DzOxLNa2.js";import{t as Y}from"./next-frame-once-qdYFoq8G.js";import{i as re,n as ie,r as X,t as ae}from"./create-Btm5lh4r.js";import{t as Z}from"./misc-DDs3MKLt.js";import{l as Q}from"./light-C4p8j3lw.js";import{a as oe,c as se,d as ce,i as le,l as ue,o as de,s as fe,t as pe,u as me}from"./Popover-D4qwnV21.js";import{a as he,i as ge}from"./text-DPkxR-eM.js";import{a as _e,f as ve,l as ye,r as be,s as xe,t as Se}from"./light-Cam34u5Q.js";import{t as Ce}from"./use-locale-DtkTcaad.js";import{n as we}from"./Input-DBVy3Lu_.js";import{t as Te}from"./Tag-B-CPkXsO.js";import{D as Ee,I as $}from"./index-DZhlw9Fw.js";function De(e){return e&-e}var Oe=class{constructor(e,t){this.l=e,this.min=t;let n=Array(e+1);for(let t=0;t<e+1;++t)n[t]=0;this.ft=n}add(e,t){if(t===0)return;let{l:n,ft:r}=this;for(e+=1;e<=n;)r[e]+=t,e+=De(e)}get(e){return this.sum(e+1)-this.sum(e)}sum(e){if(e===void 0&&(e=this.l),e<=0)return 0;let{ft:t,min:n,l:r}=this;if(e>r)throw Error("[FinweckTree.sum]: `i` is larger than length.");let i=e*n;for(;e>0;)i+=t[e],e-=De(e);return i}getBound(e){let t=0,n=this.l;for(;n>t;){let r=Math.floor((t+n)/2),i=this.sum(r);if(i>e){n=r;continue}else if(i<e){if(t===r)return this.sum(t+1)<=e?t+1:r;t=r}else return r}return t}},ke;function Ae(){return typeof document>`u`?!1:(ke===void 0&&(ke=`matchMedia`in window&&window.matchMedia(`(pointer:coarse)`).matches),ke)}var je;function Me(){return typeof document>`u`?1:(je===void 0&&(je=`chrome`in window?window.devicePixelRatio:1),je)}var Ne=`VVirtualListXScroll`;function Pe({columnsRef:e,renderColRef:t,renderItemWithColsRef:n}){let r=B(0),i=B(0),a=P(()=>{let t=e.value;if(t.length===0)return null;let n=new Oe(t.length,0);return t.forEach((e,t)=>{n.add(t,e.width)}),n});return J(Ne,{startIndexRef:s(()=>{let e=a.value;return e===null?0:Math.max(e.getBound(i.value)-1,0)}),endIndexRef:s(()=>{let t=a.value;return t===null?0:Math.min(t.getBound(i.value+r.value)+1,e.value.length-1)}),columnsRef:e,renderColRef:t,renderItemWithColsRef:n,getLeft:e=>{let t=a.value;return t===null?0:t.sum(e)}}),{listWidthRef:r,scrollLeftRef:i}}var Fe=W({name:`VirtualListRow`,props:{index:{type:Number,required:!0},item:{type:Object,required:!0}},setup(){let{startIndexRef:e,endIndexRef:t,columnsRef:n,getLeft:r,renderColRef:i,renderItemWithColsRef:a}=v(Ne);return{startIndex:e,endIndex:t,columns:n,renderCol:i,renderItemWithCols:a,getLeft:r}},render(){let{startIndex:e,endIndex:t,columns:n,renderCol:r,renderItemWithCols:i,getLeft:a,item:o}=this;if(i!=null)return i({itemIndex:this.index,startColIndex:e,endColIndex:t,allColumns:n,item:o,getLeft:a});if(r!=null){let i=[];for(let s=e;s<=t;++s){let e=n[s];i.push(r({column:e,left:a(s),item:o}))}return i}return null}}),Ie=oe(`.v-vl`,{maxHeight:`inherit`,height:`100%`,overflow:`auto`,minWidth:`1px`},[oe(`&:not(.v-vl--show-scrollbar)`,{scrollbarWidth:`none`},[oe(`&::-webkit-scrollbar, &::-webkit-scrollbar-track-piece, &::-webkit-scrollbar-thumb`,{width:0,height:0,display:`none`})])]),Le=W({name:`VirtualList`,inheritAttrs:!1,props:{showScrollbar:{type:Boolean,default:!0},columns:{type:Array,default:()=>[]},renderCol:Function,renderItemWithCols:Function,items:{type:Array,default:()=>[]},itemSize:{type:Number,required:!0},itemResizable:Boolean,itemsStyle:[String,Object],visibleItemsTag:{type:[String,Object],default:`div`},visibleItemsProps:Object,ignoreItemResize:Boolean,onScroll:Function,onWheel:Function,onResize:Function,defaultScrollKey:[Number,String],defaultScrollIndex:Number,keyField:{type:String,default:`key`},paddingTop:{type:[Number,String],default:0},paddingBottom:{type:[Number,String],default:0}},setup(e){let t=n();Ie.mount({id:`vueuc/virtual-list`,head:!0,anchorMetaName:de,ssr:t}),d(()=>{let{defaultScrollIndex:t,defaultScrollKey:n}=e;t==null?n!=null&&C({key:n}):C({index:t})});let r=!1,i=!1;h(()=>{if(r=!1,!i){i=!0;return}C({top:b.value,left:f.value})}),o(()=>{r=!0,i||=!0});let a=s(()=>{if(e.renderCol==null&&e.renderItemWithCols==null||e.columns.length===0)return;let t=0;return e.columns.forEach(e=>{t+=e.width}),t}),c=P(()=>{let t=new Map,{keyField:n}=e;return e.items.forEach((e,r)=>{t.set(e[n],r)}),t}),{scrollLeftRef:f,listWidthRef:p}=Pe({columnsRef:V(e,`columns`),renderColRef:V(e,`renderCol`),renderItemWithColsRef:V(e,`renderItemWithCols`)}),m=B(null),g=B(void 0),_=new Map,v=P(()=>{let{items:t,itemSize:n,keyField:r}=e,i=new Oe(t.length,n);return t.forEach((e,t)=>{let n=e[r],a=_.get(n);a!==void 0&&i.add(t,a)}),i}),y=B(0),b=B(0),x=s(()=>Math.max(v.value.getBound(b.value-u(e.paddingTop))-1,0)),S=P(()=>{let{value:t}=g;if(t===void 0)return[];let{items:n,itemSize:r}=e,i=x.value,a=Math.min(i+Math.ceil(t/r+1),n.length-1),o=[];for(let e=i;e<=a;++e)o.push(n[e]);return o}),C=(e,t)=>{if(typeof e==`number`){D(e,t,`auto`);return}let{left:n,top:r,index:i,key:a,position:o,behavior:s,debounce:l=!0}=e;if(n!==void 0||r!==void 0)D(n,r,s);else if(i!==void 0)E(i,s,l);else if(a!==void 0){let e=c.value.get(a);e!==void 0&&E(e,s,l)}else o===`bottom`?D(0,2**53-1,s):o===`top`&&D(0,0,s)},w,T=null;function E(t,n,r){let{value:i}=v,a=i.sum(t)+u(e.paddingTop);if(!r)m.value.scrollTo({left:0,top:a,behavior:n});else{w=t,T!==null&&window.clearTimeout(T),T=window.setTimeout(()=>{w=void 0,T=null},16);let{scrollTop:e,offsetHeight:r}=m.value;if(a>e){let o=i.get(t);a+o<=e+r||m.value.scrollTo({left:0,top:a+o-r,behavior:n})}else m.value.scrollTo({left:0,top:a,behavior:n})}}function D(e,t,n){m.value.scrollTo({left:e,top:t,behavior:n})}function ee(t,n){if(r||e.ignoreItemResize||F(n.target))return;let{value:i}=v,a=c.value.get(t),o=i.get(a),s=n.borderBoxSize?.[0]?.blockSize??n.contentRect.height;if(s===o)return;s-e.itemSize===0?_.delete(t):_.set(t,s-e.itemSize);let l=s-o;if(l===0)return;i.add(a,l);let u=m.value;if(u!=null){if(w===void 0){let e=i.sum(a);u.scrollTop>e&&u.scrollBy(0,l)}else(a<w||a===w&&s+i.sum(a)>u.scrollTop+u.offsetHeight)&&u.scrollBy(0,l);N()}y.value++}let O=!Ae(),k=!1;function A(t){var n;(n=e.onScroll)==null||n.call(e,t),(!O||!k)&&N()}function j(t){var n;if((n=e.onWheel)==null||n.call(e,t),O){let e=m.value;if(e!=null){if(t.deltaX===0&&(e.scrollTop===0&&t.deltaY<=0||e.scrollTop+e.offsetHeight>=e.scrollHeight&&t.deltaY>=0))return;t.preventDefault(),e.scrollTop+=t.deltaY/Me(),e.scrollLeft+=t.deltaX/Me(),N(),k=!0,Y(()=>{k=!1})}}}function M(t){if(r||F(t.target))return;if(e.renderCol==null&&e.renderItemWithCols==null){if(t.contentRect.height===g.value)return}else if(t.contentRect.height===g.value&&t.contentRect.width===p.value)return;g.value=t.contentRect.height,p.value=t.contentRect.width;let{onResize:n}=e;n!==void 0&&n(t)}function N(){let{value:e}=m;e!=null&&(b.value=e.scrollTop,f.value=e.scrollLeft)}function F(e){let t=e;for(;t!==null;){if(t.style.display===`none`)return!0;t=t.parentElement}return!1}return{listHeight:g,listStyle:{overflow:`auto`},keyToIndex:c,itemsStyle:P(()=>{let{itemResizable:t}=e,n=l(v.value.sum());return y.value,[e.itemsStyle,{boxSizing:`content-box`,width:l(a.value),height:t?``:n,minHeight:t?n:``,paddingTop:l(e.paddingTop),paddingBottom:l(e.paddingBottom)}]}),visibleItemsStyle:P(()=>(y.value,{transform:`translateY(${l(v.value.sum(x.value))})`})),viewportItems:S,listElRef:m,itemsElRef:B(null),scrollTo:C,handleListResize:M,handleListScroll:A,handleListWheel:j,handleItemResize:ee}},render(){let{itemResizable:e,keyField:n,keyToIndex:r,visibleItemsTag:i}=this;return a(y,{onResize:this.handleListResize},{default:()=>{var o;return a(`div`,t(this.$attrs,{class:[`v-vl`,this.showScrollbar&&`v-vl--show-scrollbar`],onScroll:this.handleListScroll,onWheel:this.handleListWheel,ref:`listElRef`}),[this.items.length===0?(o=this.$slots).empty?.call(o):a(`div`,{ref:`itemsElRef`,class:`v-vl-items`,style:this.itemsStyle},[a(i,Object.assign({class:`v-vl-visible-items`,style:this.visibleItemsStyle},this.visibleItemsProps),{default:()=>{let{renderCol:t,renderItemWithCols:i}=this;return this.viewportItems.map(o=>{let s=o[n],c=r.get(s),l=t==null?void 0:a(Fe,{index:c,item:o}),u=i==null?void 0:a(Fe,{index:c,item:o}),d=this.$slots.default({item:o,renderedCols:l,renderedItemWithCols:u,index:c})[0];return e?a(y,{key:s,onResize:e=>this.handleItemResize(s,e)},{default:()=>d}):(d.key=s,d)})}})])])}})}});function Re(e,t){t&&(d(()=>{let{value:n}=e;n&&R.registerHandler(n,t)}),w(e,(e,t)=>{t&&R.unregisterHandler(t)},{deep:!1}),x(()=>{let{value:t}=e;t&&R.unregisterHandler(t)}))}function ze(e){switch(typeof e){case`string`:return e||void 0;case`number`:return String(e);default:return}}function Be(e){let t=e.filter(e=>e!==void 0);if(t.length!==0)return t.length===1?t[0]:t=>{e.forEach(e=>{e&&e(t)})}}var Ve=W({name:`Checkmark`,render(){return a(`svg`,{xmlns:`http://www.w3.org/2000/svg`,viewBox:`0 0 16 16`},a(`g`,{fill:`none`},a(`path`,{d:`M14.046 3.486a.75.75 0 0 1-.032 1.06l-7.93 7.474a.85.85 0 0 1-1.188-.022l-2.68-2.72a.75.75 0 1 1 1.068-1.053l2.234 2.267l7.468-7.038a.75.75 0 0 1 1.06.032z`,fill:`currentColor`})))}}),He=W({name:`Empty`,render(){return a(`svg`,{viewBox:`0 0 28 28`,fill:`none`,xmlns:`http://www.w3.org/2000/svg`},a(`path`,{d:`M26 7.5C26 11.0899 23.0899 14 19.5 14C15.9101 14 13 11.0899 13 7.5C13 3.91015 15.9101 1 19.5 1C23.0899 1 26 3.91015 26 7.5ZM16.8536 4.14645C16.6583 3.95118 16.3417 3.95118 16.1464 4.14645C15.9512 4.34171 15.9512 4.65829 16.1464 4.85355L18.7929 7.5L16.1464 10.1464C15.9512 10.3417 15.9512 10.6583 16.1464 10.8536C16.3417 11.0488 16.6583 11.0488 16.8536 10.8536L19.5 8.20711L22.1464 10.8536C22.3417 11.0488 22.6583 11.0488 22.8536 10.8536C23.0488 10.6583 23.0488 10.3417 22.8536 10.1464L20.2071 7.5L22.8536 4.85355C23.0488 4.65829 23.0488 4.34171 22.8536 4.14645C22.6583 3.95118 22.3417 3.95118 22.1464 4.14645L19.5 6.79289L16.8536 4.14645Z`,fill:`currentColor`}),a(`path`,{d:`M25 22.75V12.5991C24.5572 13.0765 24.053 13.4961 23.5 13.8454V16H17.5L17.3982 16.0068C17.0322 16.0565 16.75 16.3703 16.75 16.75C16.75 18.2688 15.5188 19.5 14 19.5C12.4812 19.5 11.25 18.2688 11.25 16.75L11.2432 16.6482C11.1935 16.2822 10.8797 16 10.5 16H4.5V7.25C4.5 6.2835 5.2835 5.5 6.25 5.5H12.2696C12.4146 4.97463 12.6153 4.47237 12.865 4H6.25C4.45507 4 3 5.45507 3 7.25V22.75C3 24.5449 4.45507 26 6.25 26H21.75C23.5449 26 25 24.5449 25 22.75ZM4.5 22.75V17.5H9.81597L9.85751 17.7041C10.2905 19.5919 11.9808 21 14 21L14.215 20.9947C16.2095 20.8953 17.842 19.4209 18.184 17.5H23.5V22.75C23.5 23.7165 22.7165 24.5 21.75 24.5H6.25C5.2835 24.5 4.5 23.7165 4.5 22.75Z`,fill:`currentColor`}))}}),Ue=W({props:{onFocus:Function,onBlur:Function},setup(e){return()=>a(`div`,{style:`width: 0; height: 0`,tabindex:0,onFocus:e.onFocus,onBlur:e.onBlur})}}),We=E(`empty`,`
 display: flex;
 flex-direction: column;
 align-items: center;
 font-size: var(--n-font-size);
`,[D(`icon`,`
 width: var(--n-icon-size);
 height: var(--n-icon-size);
 font-size: var(--n-icon-size);
 line-height: var(--n-icon-size);
 color: var(--n-icon-color);
 transition:
 color .3s var(--n-bezier);
 `,[f(`+`,[D(`description`,`
 margin-top: 8px;
 `)])]),D(`description`,`
 transition: color .3s var(--n-bezier);
 color: var(--n-text-color);
 `),D(`extra`,`
 text-align: center;
 transition: color .3s var(--n-bezier);
 margin-top: 12px;
 color: var(--n-extra-text-color);
 `)]),Ge=W({name:`Empty`,props:Object.assign(Object.assign({},I.props),{description:String,showDescription:{type:Boolean,default:!0},showIcon:{type:Boolean,default:!0},size:{type:String,default:`medium`},renderIcon:Function}),slots:Object,setup(e){let{mergedClsPrefixRef:t,inlineThemeDisabled:n,mergedComponentPropsRef:r}=H(e),i=I(`Empty`,`-empty`,We,ye,e,t),{localeRef:o}=Ce(`Empty`),s=P(()=>e.description??r?.value?.Empty?.description),c=P(()=>r?.value?.Empty?.renderIcon||(()=>a(He,null))),l=P(()=>{let{size:t}=e,{common:{cubicBezierEaseInOut:n},self:{[j(`iconSize`,t)]:r,[j(`fontSize`,t)]:a,textColor:o,iconColor:s,extraTextColor:c}}=i.value;return{"--n-icon-size":r,"--n-font-size":a,"--n-bezier":n,"--n-text-color":o,"--n-icon-color":s,"--n-extra-text-color":c}}),u=n?N(`empty`,P(()=>{let t=``,{size:n}=e;return t+=n[0],t}),l,e):void 0;return{mergedClsPrefix:t,mergedRenderIcon:c,localizedDescription:P(()=>s.value||o.value.description),cssVars:n?void 0:l,themeClass:u?.themeClass,onRender:u?.onRender}},render(){let{$slots:e,mergedClsPrefix:t,onRender:n}=this;return n?.(),a(`div`,{class:[`${t}-empty`,this.themeClass],style:this.cssVars},this.showIcon?a(`div`,{class:`${t}-empty__icon`},e.icon?e.icon():a(L,{clsPrefix:t},{default:this.mergedRenderIcon})):null,this.showDescription?a(`div`,{class:`${t}-empty__description`},e.default?e.default():this.localizedDescription):null,e.extra?a(`div`,{class:`${t}-empty__extra`},e.extra()):null)}}),Ke=W({name:`NBaseSelectGroupHeader`,props:{clsPrefix:{type:String,required:!0},tmNode:{type:Object,required:!0}},setup(){let{renderLabelRef:e,renderOptionRef:t,labelFieldRef:n,nodePropsRef:r}=v(ce);return{labelField:n,nodeProps:r,renderLabel:e,renderOption:t}},render(){let{clsPrefix:e,renderLabel:t,renderOption:n,nodeProps:r,tmNode:{rawNode:i}}=this,o=r?.(i),s=t?t(i,!1):$(i[this.labelField],i,!1),c=a(`div`,Object.assign({},o,{class:[`${e}-base-select-group-header`,o?.class]}),s);return i.render?i.render({node:c,option:i}):n?n({node:c,option:i,selected:!1}):c}});function qe(e,t){return a(O,{name:`fade-in-scale-up-transition`},{default:()=>e?a(L,{clsPrefix:t,class:`${t}-base-select-option__check`},{default:()=>a(Ve)}):null})}var Je=W({name:`NBaseSelectOption`,props:{clsPrefix:{type:String,required:!0},tmNode:{type:Object,required:!0}},setup(e){let{valueRef:t,pendingTmNodeRef:n,multipleRef:r,valueSetRef:i,renderLabelRef:a,renderOptionRef:o,labelFieldRef:c,valueFieldRef:l,showCheckmarkRef:u,nodePropsRef:d,handleOptionClick:f,handleOptionMouseEnter:p}=v(ce),m=s(()=>{let{value:t}=n;return t?e.tmNode.key===t.key:!1});function h(t){let{tmNode:n}=e;n.disabled||f(t,n)}function g(t){let{tmNode:n}=e;n.disabled||p(t,n)}function _(t){let{tmNode:n}=e,{value:r}=m;n.disabled||r||p(t,n)}return{multiple:r,isGrouped:s(()=>{let{tmNode:t}=e,{parent:n}=t;return n&&n.rawNode.type===`group`}),showCheckmark:u,nodeProps:d,isPending:m,isSelected:s(()=>{let{value:n}=t,{value:a}=r;if(n===null)return!1;let o=e.tmNode.rawNode[l.value];if(a){let{value:e}=i;return e.has(o)}else return n===o}),labelField:c,renderLabel:a,renderOption:o,handleMouseMove:_,handleMouseEnter:g,handleClick:h}},render(){let{clsPrefix:e,tmNode:{rawNode:t},isSelected:n,isPending:r,isGrouped:i,showCheckmark:o,nodeProps:s,renderOption:c,renderLabel:l,handleClick:u,handleMouseEnter:d,handleMouseMove:f}=this,p=qe(n,e),m=l?[l(t,n),o&&p]:[$(t[this.labelField],t,n),o&&p],h=s?.(t),g=a(`div`,Object.assign({},h,{class:[`${e}-base-select-option`,t.class,h?.class,{[`${e}-base-select-option--disabled`]:t.disabled,[`${e}-base-select-option--selected`]:n,[`${e}-base-select-option--grouped`]:i,[`${e}-base-select-option--pending`]:r,[`${e}-base-select-option--show-checkmark`]:o}],style:[h?.style||``,t.style||``],onClick:Be([u,h?.onClick]),onMouseenter:Be([d,h?.onMouseenter]),onMousemove:Be([f,h?.onMousemove])}),a(`div`,{class:`${e}-base-select-option__content`},m));return t.render?t.render({node:g,option:t,selected:n}):c?c({node:g,option:t,selected:n}):g}}),Ye=E(`base-select-menu`,`
 line-height: 1.5;
 outline: none;
 z-index: 0;
 position: relative;
 border-radius: var(--n-border-radius);
 transition:
 background-color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier);
 background-color: var(--n-color);
`,[E(`scrollbar`,`
 max-height: var(--n-height);
 `),E(`virtual-list`,`
 max-height: var(--n-height);
 `),E(`base-select-option`,`
 min-height: var(--n-option-height);
 font-size: var(--n-option-font-size);
 display: flex;
 align-items: center;
 `,[D(`content`,`
 z-index: 1;
 white-space: nowrap;
 text-overflow: ellipsis;
 overflow: hidden;
 `)]),E(`base-select-group-header`,`
 min-height: var(--n-option-height);
 font-size: .93em;
 display: flex;
 align-items: center;
 `),E(`base-select-menu-option-wrapper`,`
 position: relative;
 width: 100%;
 `),D(`loading, empty`,`
 display: flex;
 padding: 12px 32px;
 flex: 1;
 justify-content: center;
 `),D(`loading`,`
 color: var(--n-loading-color);
 font-size: var(--n-loading-size);
 `),D(`header`,`
 padding: 8px var(--n-option-padding-left);
 font-size: var(--n-option-font-size);
 transition: 
 color .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 border-bottom: 1px solid var(--n-action-divider-color);
 color: var(--n-action-text-color);
 `),D(`action`,`
 padding: 8px var(--n-option-padding-left);
 font-size: var(--n-option-font-size);
 transition: 
 color .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 border-top: 1px solid var(--n-action-divider-color);
 color: var(--n-action-text-color);
 `),E(`base-select-group-header`,`
 position: relative;
 cursor: default;
 padding: var(--n-option-padding);
 color: var(--n-group-header-text-color);
 `),E(`base-select-option`,`
 cursor: pointer;
 position: relative;
 padding: var(--n-option-padding);
 transition:
 color .3s var(--n-bezier),
 opacity .3s var(--n-bezier);
 box-sizing: border-box;
 color: var(--n-option-text-color);
 opacity: 1;
 `,[C(`show-checkmark`,`
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
 `),C(`grouped`,`
 padding-left: calc(var(--n-option-padding-left) * 1.5);
 `),C(`pending`,[f(`&::before`,`
 background-color: var(--n-option-color-pending);
 `)]),C(`selected`,`
 color: var(--n-option-text-color-active);
 `,[f(`&::before`,`
 background-color: var(--n-option-color-active);
 `),C(`pending`,[f(`&::before`,`
 background-color: var(--n-option-color-active-pending);
 `)])]),C(`disabled`,`
 cursor: not-allowed;
 `,[e(`selected`,`
 color: var(--n-option-text-color-disabled);
 `),C(`selected`,`
 opacity: var(--n-option-opacity-disabled);
 `)]),D(`check`,`
 font-size: 16px;
 position: absolute;
 right: calc(var(--n-option-padding-right) - 4px);
 top: calc(50% - 7px);
 color: var(--n-option-check-color);
 transition: color .3s var(--n-bezier);
 `,[Ee({enterScale:`0.5`})])])]),Xe=W({name:`InternalSelectMenu`,props:Object.assign(Object.assign({},I.props),{clsPrefix:{type:String,required:!0},scrollable:{type:Boolean,default:!0},treeMate:{type:Object,required:!0},multiple:Boolean,size:{type:String,default:`medium`},value:{type:[String,Number,Array],default:null},autoPending:Boolean,virtualScroll:{type:Boolean,default:!0},show:{type:Boolean,default:!0},labelField:{type:String,default:`label`},valueField:{type:String,default:`value`},loading:Boolean,focusable:Boolean,renderLabel:Function,renderOption:Function,nodeProps:Function,showCheckmark:{type:Boolean,default:!0},onMousedown:Function,onScroll:Function,onFocus:Function,onBlur:Function,onKeyup:Function,onKeydown:Function,onTabOut:Function,onMouseenter:Function,onMouseleave:Function,onResize:Function,resetMenuOnOptionsChange:{type:Boolean,default:!0},inlineThemeDisabled:Boolean,scrollbarProps:Object,onToggle:Function}),setup(e){let{mergedClsPrefixRef:t,mergedRtlRef:n,mergedComponentPropsRef:r}=H(e),i=b(`InternalSelectMenu`,n,t),a=I(`InternalSelectMenu`,`-internal-select-menu`,Ye,xe,e,V(e,`clsPrefix`)),o=B(null),s=B(null),c=B(null),l=P(()=>e.treeMate.getFlattenedNodes()),f=P(()=>ie(l.value)),p=B(null);function m(){let{treeMate:t}=e,n=null,{value:r}=e;r===null?n=t.getFirstAvailableNode():(n=e.multiple?t.getNode((r||[])[(r||[]).length-1]):t.getNode(r),(!n||n.disabled)&&(n=t.getFirstAvailableNode())),U(n||null)}function h(){let{value:t}=p;t&&!e.treeMate.getNode(t.key)&&(p.value=null)}let g;w(()=>e.show,t=>{t?g=w(()=>e.treeMate,()=>{e.resetMenuOnOptionsChange?(e.autoPending?m():h(),te(W)):h()},{immediate:!0}):g?.()},{immediate:!0}),x(()=>{g?.()});let _=P(()=>u(a.value.self[j(`optionHeight`,e.size)])),v=P(()=>T(a.value.self[j(`padding`,e.size)])),y=P(()=>e.multiple&&Array.isArray(e.value)?new Set(e.value):new Set),S=P(()=>{let e=l.value;return e&&e.length===0}),C=P(()=>r?.value?.Select?.renderEmpty);function E(t){let{onToggle:n}=e;n&&n(t)}function D(t){let{onScroll:n}=e;n&&n(t)}function ee(e){var t;(t=c.value)==null||t.sync(),D(e)}function O(){var e;(e=c.value)==null||e.sync()}function k(){let{value:e}=p;return e||null}function A(e,t){t.disabled||U(t,!1)}function M(e,t){t.disabled||E(t)}function F(t){var n;re(t,`action`)||(n=e.onKeyup)==null||n.call(e,t)}function L(t){var n;re(t,`action`)||(n=e.onKeydown)==null||n.call(e,t)}function R(t){var n;(n=e.onMousedown)==null||n.call(e,t),!e.focusable&&t.preventDefault()}function z(){let{value:e}=p;e&&U(e.getNext({loop:!0}),!0)}function ne(){let{value:e}=p;e&&U(e.getPrev({loop:!0}),!0)}function U(e,t=!1){p.value=e,t&&W()}function W(){var t,n;let r=p.value;if(!r)return;let i=f.value(r.key);i!==null&&(e.virtualScroll?(t=s.value)==null||t.scrollTo({index:i}):(n=c.value)==null||n.scrollTo({index:i,elSize:_.value}))}function G(t){var n;o.value?.contains(t.target)&&((n=e.onFocus)==null||n.call(e,t))}function K(t){var n;o.value?.contains(t.relatedTarget)||(n=e.onBlur)==null||n.call(e,t)}J(ce,{handleOptionMouseEnter:A,handleOptionClick:M,valueSetRef:y,pendingTmNodeRef:p,nodePropsRef:V(e,`nodeProps`),showCheckmarkRef:V(e,`showCheckmark`),multipleRef:V(e,`multiple`),valueRef:V(e,`value`),renderLabelRef:V(e,`renderLabel`),renderOptionRef:V(e,`renderOption`),labelFieldRef:V(e,`labelField`),valueFieldRef:V(e,`valueField`)}),J(me,o),d(()=>{let{value:e}=c;e&&e.sync()});let q=P(()=>{let{size:t}=e,{common:{cubicBezierEaseInOut:n},self:{height:r,borderRadius:i,color:o,groupHeaderTextColor:s,actionDividerColor:c,optionTextColorPressed:l,optionTextColor:u,optionTextColorDisabled:d,optionTextColorActive:f,optionOpacityDisabled:p,optionCheckColor:m,actionTextColor:h,optionColorPending:g,optionColorActive:_,loadingColor:v,loadingSize:y,optionColorActivePending:b,[j(`optionFontSize`,t)]:x,[j(`optionHeight`,t)]:S,[j(`optionPadding`,t)]:C}}=a.value;return{"--n-height":r,"--n-action-divider-color":c,"--n-action-text-color":h,"--n-bezier":n,"--n-border-radius":i,"--n-color":o,"--n-option-font-size":x,"--n-group-header-text-color":s,"--n-option-check-color":m,"--n-option-color-pending":g,"--n-option-color-active":_,"--n-option-color-active-pending":b,"--n-option-height":S,"--n-option-opacity-disabled":p,"--n-option-text-color":u,"--n-option-text-color-active":f,"--n-option-text-color-disabled":d,"--n-option-text-color-pressed":l,"--n-option-padding":C,"--n-option-padding-left":T(C,`left`),"--n-option-padding-right":T(C,`right`),"--n-loading-color":v,"--n-loading-size":y}}),{inlineThemeDisabled:Y}=e,X=Y?N(`internal-select-menu`,P(()=>e.size[0]),q,e):void 0,ae={selfRef:o,next:z,prev:ne,getPendingTmNode:k};return Re(o,e.onResize),Object.assign({mergedTheme:a,mergedClsPrefix:t,rtlEnabled:i,virtualListRef:s,scrollbarRef:c,itemSize:_,padding:v,flattenedNodes:l,empty:S,mergedRenderEmpty:C,virtualListContainer(){let{value:e}=s;return e?.listElRef},virtualListContent(){let{value:e}=s;return e?.itemsElRef},doScroll:D,handleFocusin:G,handleFocusout:K,handleKeyUp:F,handleKeyDown:L,handleMouseDown:R,handleVirtualListResize:O,handleVirtualListScroll:ee,cssVars:Y?void 0:q,themeClass:X?.themeClass,onRender:X?.onRender},ae)},render(){let{$slots:e,virtualScroll:t,clsPrefix:n,mergedTheme:i,themeClass:o,onRender:s}=this;return s?.(),a(`div`,{ref:`selfRef`,tabindex:this.focusable?0:-1,class:[`${n}-base-select-menu`,`${n}-base-select-menu--${this.size}-size`,this.rtlEnabled&&`${n}-base-select-menu--rtl`,o,this.multiple&&`${n}-base-select-menu--multiple`],style:this.cssVars,onFocusin:this.handleFocusin,onFocusout:this.handleFocusout,onKeyup:this.handleKeyUp,onKeydown:this.handleKeyDown,onMousedown:this.handleMouseDown,onMouseenter:this.onMouseenter,onMouseleave:this.onMouseleave},K(e.header,e=>e&&a(`div`,{class:`${n}-base-select-menu__header`,"data-header":!0,key:`header`},e)),this.loading?a(`div`,{class:`${n}-base-select-menu__loading`},a(r,{clsPrefix:n,strokeWidth:20})):this.empty?a(`div`,{class:`${n}-base-select-menu__empty`,"data-empty":!0},q(e.empty,()=>[this.mergedRenderEmpty?.call(this)||a(Ge,{theme:i.peers.Empty,themeOverrides:i.peerOverrides.Empty,size:this.size})])):a(ee,Object.assign({ref:`scrollbarRef`,theme:i.peers.Scrollbar,themeOverrides:i.peerOverrides.Scrollbar,scrollable:this.scrollable,container:t?this.virtualListContainer:void 0,content:t?this.virtualListContent:void 0,onScroll:t?void 0:this.doScroll},this.scrollbarProps),{default:()=>t?a(Le,{ref:`virtualListRef`,class:`${n}-virtual-list`,items:this.flattenedNodes,itemSize:this.itemSize,showScrollbar:!1,paddingTop:this.padding.top,paddingBottom:this.padding.bottom,onResize:this.handleVirtualListResize,onScroll:this.handleVirtualListScroll,itemResizable:!0},{default:({item:e})=>e.isGroup?a(Ke,{key:e.key,clsPrefix:n,tmNode:e}):e.ignored?null:a(Je,{clsPrefix:n,key:e.key,tmNode:e})}):a(`div`,{class:`${n}-base-select-menu-option-wrapper`,style:{paddingTop:this.padding.top,paddingBottom:this.padding.bottom}},this.flattenedNodes.map(e=>e.isGroup?a(Ke,{key:e.key,clsPrefix:n,tmNode:e}):a(Je,{clsPrefix:n,key:e.key,tmNode:e})))}),K(e.action,e=>e&&[a(`div`,{class:`${n}-base-select-menu__action`,"data-action":!0,key:`action`},e),a(Ue,{onFocus:this.onTabOut,key:`focus-detector`})]))}}),Ze=f([E(`base-selection`,`
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
 `,[E(`base-loading`,`
 color: var(--n-loading-color);
 `),E(`base-selection-tags`,`min-height: var(--n-height);`),D(`border, state-border`,`
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
 `),D(`state-border`,`
 z-index: 1;
 border-color: #0000;
 `),E(`base-suffix`,`
 cursor: pointer;
 position: absolute;
 top: 50%;
 transform: translateY(-50%);
 right: 10px;
 `,[D(`arrow`,`
 font-size: var(--n-arrow-size);
 color: var(--n-arrow-color);
 transition: color .3s var(--n-bezier);
 `)]),E(`base-selection-overlay`,`
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
 `,[D(`wrapper`,`
 flex-basis: 0;
 flex-grow: 1;
 overflow: hidden;
 text-overflow: ellipsis;
 `)]),E(`base-selection-placeholder`,`
 color: var(--n-placeholder-color);
 `,[D(`inner`,`
 max-width: 100%;
 overflow: hidden;
 `)]),E(`base-selection-tags`,`
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
 `),E(`base-selection-label`,`
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
 `,[E(`base-selection-input`,`
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
 `,[D(`content`,`
 text-overflow: ellipsis;
 overflow: hidden;
 white-space: nowrap; 
 `)]),D(`render-label`,`
 color: var(--n-text-color);
 `)]),e(`disabled`,[f(`&:hover`,[D(`state-border`,`
 box-shadow: var(--n-box-shadow-hover);
 border: var(--n-border-hover);
 `)]),C(`focus`,[D(`state-border`,`
 box-shadow: var(--n-box-shadow-focus);
 border: var(--n-border-focus);
 `)]),C(`active`,[D(`state-border`,`
 box-shadow: var(--n-box-shadow-active);
 border: var(--n-border-active);
 `),E(`base-selection-label`,`background-color: var(--n-color-active);`),E(`base-selection-tags`,`background-color: var(--n-color-active);`)])]),C(`disabled`,`cursor: not-allowed;`,[D(`arrow`,`
 color: var(--n-arrow-color-disabled);
 `),E(`base-selection-label`,`
 cursor: not-allowed;
 background-color: var(--n-color-disabled);
 `,[E(`base-selection-input`,`
 cursor: not-allowed;
 color: var(--n-text-color-disabled);
 `),D(`render-label`,`
 color: var(--n-text-color-disabled);
 `)]),E(`base-selection-tags`,`
 cursor: not-allowed;
 background-color: var(--n-color-disabled);
 `),E(`base-selection-placeholder`,`
 cursor: not-allowed;
 color: var(--n-placeholder-color-disabled);
 `)]),E(`base-selection-input-tag`,`
 height: calc(var(--n-height) - 6px);
 line-height: calc(var(--n-height) - 6px);
 outline: none;
 display: none;
 position: relative;
 margin-bottom: 3px;
 max-width: 100%;
 vertical-align: bottom;
 `,[D(`input`,`
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
 `),D(`mirror`,`
 position: absolute;
 left: 0;
 top: 0;
 white-space: pre;
 visibility: hidden;
 user-select: none;
 -webkit-user-select: none;
 opacity: 0;
 `)]),[`warning`,`error`].map(t=>C(`${t}-status`,[D(`state-border`,`border: var(--n-border-${t});`),e(`disabled`,[f(`&:hover`,[D(`state-border`,`
 box-shadow: var(--n-box-shadow-hover-${t});
 border: var(--n-border-hover-${t});
 `)]),C(`active`,[D(`state-border`,`
 box-shadow: var(--n-box-shadow-active-${t});
 border: var(--n-border-active-${t});
 `),E(`base-selection-label`,`background-color: var(--n-color-active-${t});`),E(`base-selection-tags`,`background-color: var(--n-color-active-${t});`)]),C(`focus`,[D(`state-border`,`
 box-shadow: var(--n-box-shadow-focus-${t});
 border: var(--n-border-focus-${t});
 `)])])]))]),E(`base-selection-popover`,`
 margin-bottom: -3px;
 display: flex;
 flex-wrap: wrap;
 margin-right: -8px;
 `),E(`base-selection-tag-wrapper`,`
 max-width: 100%;
 display: inline-flex;
 padding: 0 7px 3px 0;
 `,[f(`&:last-child`,`padding-right: 0;`),E(`tag`,`
 font-size: 14px;
 max-width: 100%;
 `,[D(`content`,`
 line-height: 1.25;
 text-overflow: ellipsis;
 overflow: hidden;
 `)])])]),Qe=W({name:`InternalSelection`,props:Object.assign(Object.assign({},I.props),{clsPrefix:{type:String,required:!0},bordered:{type:Boolean,default:void 0},active:Boolean,pattern:{type:String,default:``},placeholder:String,selectedOption:{type:Object,default:null},selectedOptions:{type:Array,default:null},labelField:{type:String,default:`label`},valueField:{type:String,default:`value`},multiple:Boolean,filterable:Boolean,clearable:Boolean,disabled:Boolean,size:{type:String,default:`medium`},loading:Boolean,autofocus:Boolean,showArrow:{type:Boolean,default:!0},inputProps:Object,focused:Boolean,renderTag:Function,onKeydown:Function,onClick:Function,onBlur:Function,onFocus:Function,onDeleteOption:Function,maxTagCount:[String,Number],ellipsisTagPopoverProps:Object,onClear:Function,onPatternInput:Function,onPatternFocus:Function,onPatternBlur:Function,renderLabel:Function,status:String,inlineThemeDisabled:Boolean,ignoreComposition:{type:Boolean,default:!0},onResize:Function}),setup(e){let{mergedClsPrefixRef:t,mergedRtlRef:n}=H(e),r=b(`InternalSelection`,n,t),i=B(null),a=B(null),o=B(null),s=B(null),l=B(null),u=B(null),f=B(null),p=B(null),m=B(null),h=B(null),g=B(!1),_=B(!1),v=B(!1),y=I(`InternalSelection`,`-internal-selection`,Ze,_e,e,V(e,`clsPrefix`)),x=P(()=>e.clearable&&!e.disabled&&(v.value||e.active)),S=P(()=>e.selectedOption?e.renderTag?e.renderTag({option:e.selectedOption,handleClose:()=>{}}):e.renderLabel?e.renderLabel(e.selectedOption,!0):$(e.selectedOption[e.labelField],e.selectedOption,!0):e.placeholder),C=P(()=>{let t=e.selectedOption;if(t)return t[e.labelField]}),E=P(()=>e.multiple?!!(Array.isArray(e.selectedOptions)&&e.selectedOptions.length):e.selectedOption!==null);function D(){var t;let{value:n}=i;if(n){let{value:r}=a;r&&(r.style.width=`${n.offsetWidth}px`,e.maxTagCount!==`responsive`&&((t=m.value)==null||t.sync({showAllItemsBeforeCalculate:!1})))}}function ee(){let{value:e}=h;e&&(e.style.display=`none`)}function O(){let{value:e}=h;e&&(e.style.display=`inline-block`)}w(V(e,`active`),e=>{e||ee()}),w(V(e,`pattern`),()=>{e.multiple&&te(D)});function k(t){let{onFocus:n}=e;n&&n(t)}function A(t){let{onBlur:n}=e;n&&n(t)}function M(t){let{onDeleteOption:n}=e;n&&n(t)}function F(t){let{onClear:n}=e;n&&n(t)}function L(t){let{onPatternInput:n}=e;n&&n(t)}function R(e){(!e.relatedTarget||!o.value?.contains(e.relatedTarget))&&k(e)}function z(e){o.value?.contains(e.relatedTarget)||A(e)}function ne(e){F(e)}function U(){v.value=!0}function W(){v.value=!1}function G(t){!e.active||!e.filterable||t.target!==a.value&&t.preventDefault()}function K(e){M(e)}let q=B(!1);function J(t){if(t.key===`Backspace`&&!q.value&&!e.pattern.length){let{selectedOptions:t}=e;t?.length&&K(t[t.length-1])}}let Y=null;function re(t){let{value:n}=i;n&&(n.textContent=t.target.value,D()),e.ignoreComposition&&q.value?Y=t:L(t)}function ie(){q.value=!0}function X(){q.value=!1,e.ignoreComposition&&L(Y),Y=null}function ae(t){var n;_.value=!0,(n=e.onPatternFocus)==null||n.call(e,t)}function Z(t){var n;_.value=!1,(n=e.onPatternBlur)==null||n.call(e,t)}function Q(){var t,n;if(e.filterable)_.value=!1,(t=u.value)==null||t.blur(),(n=a.value)==null||n.blur();else if(e.multiple){let{value:e}=s;e?.blur()}else{let{value:e}=l;e?.blur()}}function oe(){var t,n,r;e.filterable?(_.value=!1,(t=u.value)==null||t.focus()):e.multiple?(n=s.value)==null||n.focus():(r=l.value)==null||r.focus()}function se(){let{value:e}=a;e&&(O(),e.focus())}function ce(){let{value:e}=a;e&&e.blur()}function le(e){let{value:t}=f;t&&t.setTextContent(`+${e}`)}function ue(){let{value:e}=p;return e}function de(){return a.value}let fe=null;function pe(){fe!==null&&window.clearTimeout(fe)}function me(){e.active||(pe(),fe=window.setTimeout(()=>{E.value&&(g.value=!0)},100))}function he(){pe()}function ge(e){e||(pe(),g.value=!1)}w(E,e=>{e||(g.value=!1)}),d(()=>{c(()=>{let t=u.value;t&&(e.disabled?t.removeAttribute(`tabindex`):t.tabIndex=_.value?-1:0)})}),Re(o,e.onResize);let{inlineThemeDisabled:ve}=e,ye=P(()=>{let{size:t}=e,{common:{cubicBezierEaseInOut:n},self:{fontWeight:r,borderRadius:i,color:a,placeholderColor:o,textColor:s,paddingSingle:c,paddingMultiple:l,caretColor:u,colorDisabled:d,textColorDisabled:f,placeholderColorDisabled:p,colorActive:m,boxShadowFocus:h,boxShadowActive:g,boxShadowHover:_,border:v,borderFocus:b,borderHover:x,borderActive:S,arrowColor:C,arrowColorDisabled:w,loadingColor:E,colorActiveWarning:D,boxShadowFocusWarning:ee,boxShadowActiveWarning:O,boxShadowHoverWarning:k,borderWarning:A,borderFocusWarning:M,borderHoverWarning:N,borderActiveWarning:P,colorActiveError:F,boxShadowFocusError:I,boxShadowActiveError:te,boxShadowHoverError:L,borderError:R,borderFocusError:z,borderHoverError:B,borderActiveError:V,clearColor:H,clearColorHover:ne,clearColorPressed:U,clearSize:W,arrowSize:G,[j(`height`,t)]:K,[j(`fontSize`,t)]:q}}=y.value,J=T(c),Y=T(l);return{"--n-bezier":n,"--n-border":v,"--n-border-active":S,"--n-border-focus":b,"--n-border-hover":x,"--n-border-radius":i,"--n-box-shadow-active":g,"--n-box-shadow-focus":h,"--n-box-shadow-hover":_,"--n-caret-color":u,"--n-color":a,"--n-color-active":m,"--n-color-disabled":d,"--n-font-size":q,"--n-height":K,"--n-padding-single-top":J.top,"--n-padding-multiple-top":Y.top,"--n-padding-single-right":J.right,"--n-padding-multiple-right":Y.right,"--n-padding-single-left":J.left,"--n-padding-multiple-left":Y.left,"--n-padding-single-bottom":J.bottom,"--n-padding-multiple-bottom":Y.bottom,"--n-placeholder-color":o,"--n-placeholder-color-disabled":p,"--n-text-color":s,"--n-text-color-disabled":f,"--n-arrow-color":C,"--n-arrow-color-disabled":w,"--n-loading-color":E,"--n-color-active-warning":D,"--n-box-shadow-focus-warning":ee,"--n-box-shadow-active-warning":O,"--n-box-shadow-hover-warning":k,"--n-border-warning":A,"--n-border-focus-warning":M,"--n-border-hover-warning":N,"--n-border-active-warning":P,"--n-color-active-error":F,"--n-box-shadow-focus-error":I,"--n-box-shadow-active-error":te,"--n-box-shadow-hover-error":L,"--n-border-error":R,"--n-border-focus-error":z,"--n-border-hover-error":B,"--n-border-active-error":V,"--n-clear-size":W,"--n-clear-color":H,"--n-clear-color-hover":ne,"--n-clear-color-pressed":U,"--n-arrow-size":G,"--n-font-weight":r}}),be=ve?N(`internal-selection`,P(()=>e.size[0]),ye,e):void 0;return{mergedTheme:y,mergedClearable:x,mergedClsPrefix:t,rtlEnabled:r,patternInputFocused:_,filterablePlaceholder:S,label:C,selected:E,showTagsPanel:g,isComposing:q,counterRef:f,counterWrapperRef:p,patternInputMirrorRef:i,patternInputRef:a,selfRef:o,multipleElRef:s,singleElRef:l,patternInputWrapperRef:u,overflowRef:m,inputTagElRef:h,handleMouseDown:G,handleFocusin:R,handleClear:ne,handleMouseEnter:U,handleMouseLeave:W,handleDeleteOption:K,handlePatternKeyDown:J,handlePatternInputInput:re,handlePatternInputBlur:Z,handlePatternInputFocus:ae,handleMouseEnterCounter:me,handleMouseLeaveCounter:he,handleFocusout:z,handleCompositionEnd:X,handleCompositionStart:ie,onPopoverUpdateShow:ge,focus:oe,focusInput:se,blur:Q,blurInput:ce,updateCounter:le,getCounter:ue,getTail:de,renderLabel:e.renderLabel,cssVars:ve?void 0:ye,themeClass:be?.themeClass,onRender:be?.onRender}},render(){let{status:e,multiple:t,size:n,disabled:r,filterable:i,maxTagCount:o,bordered:s,clsPrefix:c,ellipsisTagPopoverProps:l,onRender:u,renderTag:d,renderLabel:f}=this;u?.();let p=o===`responsive`,m=typeof o==`number`,h=p||m,g=a(F,null,{default:()=>a(we,{clsPrefix:c,loading:this.loading,showArrow:this.showArrow,showClear:this.mergedClearable&&this.selected,onClear:this.handleClear},{default:()=>{var e;return(e=this.$slots).arrow?.call(e)}})}),_;if(t){let{labelField:e}=this,t=t=>a(`div`,{class:`${c}-base-selection-tag-wrapper`,key:t.value},d?d({option:t,handleClose:()=>{this.handleDeleteOption(t)}}):a(Te,{size:n,closable:!t.disabled,disabled:r,onClose:()=>{this.handleDeleteOption(t)},internalCloseIsButtonTag:!1,internalCloseFocusable:!1},{default:()=>f?f(t,!0):$(t[e],t,!0)})),s=()=>(m?this.selectedOptions.slice(0,o):this.selectedOptions).map(t),u=i?a(`div`,{class:`${c}-base-selection-input-tag`,ref:`inputTagElRef`,key:`__input-tag__`},a(`input`,Object.assign({},this.inputProps,{ref:`patternInputRef`,tabindex:-1,disabled:r,value:this.pattern,autofocus:this.autofocus,class:`${c}-base-selection-input-tag__input`,onBlur:this.handlePatternInputBlur,onFocus:this.handlePatternInputFocus,onKeydown:this.handlePatternKeyDown,onInput:this.handlePatternInputInput,onCompositionstart:this.handleCompositionStart,onCompositionend:this.handleCompositionEnd})),a(`span`,{ref:`patternInputMirrorRef`,class:`${c}-base-selection-input-tag__mirror`},this.pattern)):null,v=p?()=>a(`div`,{class:`${c}-base-selection-tag-wrapper`,ref:`counterWrapperRef`},a(Te,{size:n,ref:`counterRef`,onMouseenter:this.handleMouseEnterCounter,onMouseleave:this.handleMouseLeaveCounter,disabled:r})):void 0,y;if(m){let e=this.selectedOptions.length-o;e>0&&(y=a(`div`,{class:`${c}-base-selection-tag-wrapper`,key:`__counter__`},a(Te,{size:n,ref:`counterRef`,onMouseenter:this.handleMouseEnterCounter,disabled:r},{default:()=>`+${e}`})))}let b=p?i?a(X,{ref:`overflowRef`,updateCounter:this.updateCounter,getCounter:this.getCounter,getTail:this.getTail,style:{width:`100%`,display:`flex`,overflow:`hidden`}},{default:s,counter:v,tail:()=>u}):a(X,{ref:`overflowRef`,updateCounter:this.updateCounter,getCounter:this.getCounter,style:{width:`100%`,display:`flex`,overflow:`hidden`}},{default:s,counter:v}):m&&y?s().concat(y):s(),x=h?()=>a(`div`,{class:`${c}-base-selection-popover`},p?s():this.selectedOptions.map(t)):void 0,S=h?Object.assign({show:this.showTagsPanel,trigger:`hover`,overlap:!0,placement:`top`,width:`trigger`,onUpdateShow:this.onPopoverUpdateShow,theme:this.mergedTheme.peers.Popover,themeOverrides:this.mergedTheme.peerOverrides.Popover},l):null,C=!this.selected&&(!this.active||!this.pattern&&!this.isComposing)?a(`div`,{class:`${c}-base-selection-placeholder ${c}-base-selection-overlay`},a(`div`,{class:`${c}-base-selection-placeholder__inner`},this.placeholder)):null,w=i?a(`div`,{ref:`patternInputWrapperRef`,class:`${c}-base-selection-tags`},b,p?null:u,g):a(`div`,{ref:`multipleElRef`,class:`${c}-base-selection-tags`,tabindex:r?void 0:0},b,g);_=a(M,null,h?a(pe,Object.assign({},S,{scrollable:!0,style:`max-height: calc(var(--v-target-height) * 6.6);`}),{trigger:()=>w,default:x}):w,C)}else if(i){let e=this.pattern||this.isComposing,t=this.active?!e:!this.selected,n=!this.active&&this.selected;_=a(`div`,{ref:`patternInputWrapperRef`,class:`${c}-base-selection-label`,title:this.patternInputFocused?void 0:ze(this.label)},a(`input`,Object.assign({},this.inputProps,{ref:`patternInputRef`,class:`${c}-base-selection-input`,value:this.active?this.pattern:``,placeholder:``,readonly:r,disabled:r,tabindex:-1,autofocus:this.autofocus,onFocus:this.handlePatternInputFocus,onBlur:this.handlePatternInputBlur,onInput:this.handlePatternInputInput,onCompositionstart:this.handleCompositionStart,onCompositionend:this.handleCompositionEnd})),n?a(`div`,{class:`${c}-base-selection-label__render-label ${c}-base-selection-overlay`,key:`input`},a(`div`,{class:`${c}-base-selection-overlay__wrapper`},d?d({option:this.selectedOption,handleClose:()=>{}}):f?f(this.selectedOption,!0):$(this.label,this.selectedOption,!0))):null,t?a(`div`,{class:`${c}-base-selection-placeholder ${c}-base-selection-overlay`,key:`placeholder`},a(`div`,{class:`${c}-base-selection-overlay__wrapper`},this.filterablePlaceholder)):null,g)}else _=a(`div`,{ref:`singleElRef`,class:`${c}-base-selection-label`,tabindex:this.disabled?void 0:0},this.label===void 0?a(`div`,{class:`${c}-base-selection-placeholder ${c}-base-selection-overlay`,key:`placeholder`},a(`div`,{class:`${c}-base-selection-placeholder__inner`},this.placeholder)):a(`div`,{class:`${c}-base-selection-input`,title:ze(this.label),key:`input`},a(`div`,{class:`${c}-base-selection-input__content`},d?d({option:this.selectedOption,handleClose:()=>{}}):f?f(this.selectedOption,!0):$(this.label,this.selectedOption,!0))),g);return a(`div`,{ref:`selfRef`,class:[`${c}-base-selection`,this.rtlEnabled&&`${c}-base-selection--rtl`,this.themeClass,e&&`${c}-base-selection--${e}-status`,{[`${c}-base-selection--active`]:this.active,[`${c}-base-selection--selected`]:this.selected||this.active&&this.pattern,[`${c}-base-selection--disabled`]:this.disabled,[`${c}-base-selection--multiple`]:this.multiple,[`${c}-base-selection--focus`]:this.focused}],style:this.cssVars,onClick:this.onClick,onMouseenter:this.handleMouseEnter,onMouseleave:this.handleMouseLeave,onKeydown:this.onKeydown,onFocusin:this.handleFocusin,onFocusout:this.handleFocusout,onMousedown:this.handleMouseDown},_,s?a(`div`,{class:`${c}-base-selection__border`}):null,s?a(`div`,{class:`${c}-base-selection__state-border`}):null)}});function $e(e){return e.type===`group`}function et(e){return e.type===`ignored`}function tt(e,t){try{return!!(1+t.toString().toLowerCase().indexOf(e.trim().toLowerCase()))}catch{return!1}}function nt(e,t){return{getIsGroup:$e,getIgnored:et,getKey(t){return $e(t)?t.name||t.key||`key-required`:t[e]},getChildren(e){return e[t]}}}function rt(e,t,n,r){if(!t)return e;function i(e){if(!Array.isArray(e))return[];let a=[];for(let o of e)if($e(o)){let e=i(o[r]);e.length&&a.push(Object.assign({},o,{[r]:e}))}else if(et(o))continue;else t(n,o)&&a.push(o);return a}return i(e)}function it(e,t,n){let r=new Map;return e.forEach(e=>{$e(e)?e[n].forEach(e=>{r.set(e[t],e)}):r.set(e[t],e)}),r}var at=g(`n-checkbox-group`),ot=W({name:`CheckboxGroup`,props:{min:Number,max:Number,size:String,value:Array,defaultValue:{type:Array,default:null},disabled:{type:Boolean,default:void 0},"onUpdate:value":[Function,Array],onUpdateValue:[Function,Array],onChange:[Function,Array]},setup(e){let{mergedClsPrefixRef:t}=H(e),n=A(e),{mergedSizeRef:r,mergedDisabledRef:i}=n,a=B(e.defaultValue),o=he(P(()=>e.value),a),s=P(()=>o.value?.length||0),c=P(()=>Array.isArray(o.value)?new Set(o.value):new Set);function l(t,r){let{nTriggerFormInput:i,nTriggerFormChange:s}=n,{onChange:c,"onUpdate:value":l,onUpdateValue:u}=e;if(Array.isArray(o.value)){let e=Array.from(o.value),n=e.findIndex(e=>e===r);t?~n||(e.push(r),u&&G(u,e,{actionType:`check`,value:r}),l&&G(l,e,{actionType:`check`,value:r}),i(),s(),a.value=e,c&&G(c,e)):~n&&(e.splice(n,1),u&&G(u,e,{actionType:`uncheck`,value:r}),l&&G(l,e,{actionType:`uncheck`,value:r}),c&&G(c,e),a.value=e,i(),s())}else t?(u&&G(u,[r],{actionType:`check`,value:r}),l&&G(l,[r],{actionType:`check`,value:r}),c&&G(c,[r]),a.value=[r],i(),s()):(u&&G(u,[],{actionType:`uncheck`,value:r}),l&&G(l,[],{actionType:`uncheck`,value:r}),c&&G(c,[]),a.value=[],i(),s())}return J(at,{checkedCountRef:s,maxRef:V(e,`max`),minRef:V(e,`min`),valueSetRef:c,disabledRef:i,mergedSizeRef:r,toggleCheckbox:l}),{mergedClsPrefix:t}},render(){return a(`div`,{class:`${this.mergedClsPrefix}-checkbox-group`,role:`group`},this.$slots)}}),st=()=>a(`svg`,{viewBox:`0 0 64 64`,class:`check-icon`},a(`path`,{d:`M50.42,16.76L22.34,39.45l-8.1-11.46c-1.12-1.58-3.3-1.96-4.88-0.84c-1.58,1.12-1.95,3.3-0.84,4.88l10.26,14.51  c0.56,0.79,1.42,1.31,2.38,1.45c0.16,0.02,0.32,0.03,0.48,0.03c0.8,0,1.57-0.27,2.2-0.78l30.99-25.03c1.5-1.21,1.74-3.42,0.52-4.92  C54.13,15.78,51.93,15.55,50.42,16.76z`})),ct=()=>a(`svg`,{viewBox:`0 0 100 100`,class:`line-icon`},a(`path`,{d:`M80.2,55.5H21.4c-2.8,0-5.1-2.5-5.1-5.5l0,0c0-3,2.3-5.5,5.1-5.5h58.7c2.8,0,5.1,2.5,5.1,5.5l0,0C85.2,53.1,82.9,55.5,80.2,55.5z`})),lt=f([E(`checkbox`,`
 font-size: var(--n-font-size);
 outline: none;
 cursor: pointer;
 display: inline-flex;
 flex-wrap: nowrap;
 align-items: flex-start;
 word-break: break-word;
 line-height: var(--n-size);
 --n-merged-color-table: var(--n-color-table);
 `,[C(`show-label`,`line-height: var(--n-label-line-height);`),f(`&:hover`,[E(`checkbox-box`,[D(`border`,`border: var(--n-border-checked);`)])]),f(`&:focus:not(:active)`,[E(`checkbox-box`,[D(`border`,`
 border: var(--n-border-focus);
 box-shadow: var(--n-box-shadow-focus);
 `)])]),C(`inside-table`,[E(`checkbox-box`,`
 background-color: var(--n-merged-color-table);
 `)]),C(`checked`,[E(`checkbox-box`,`
 background-color: var(--n-color-checked);
 `,[E(`checkbox-icon`,[f(`.check-icon`,`
 opacity: 1;
 transform: scale(1);
 `)])])]),C(`indeterminate`,[E(`checkbox-box`,[E(`checkbox-icon`,[f(`.check-icon`,`
 opacity: 0;
 transform: scale(.5);
 `),f(`.line-icon`,`
 opacity: 1;
 transform: scale(1);
 `)])])]),C(`checked, indeterminate`,[f(`&:focus:not(:active)`,[E(`checkbox-box`,[D(`border`,`
 border: var(--n-border-checked);
 box-shadow: var(--n-box-shadow-focus);
 `)])]),E(`checkbox-box`,`
 background-color: var(--n-color-checked);
 border-left: 0;
 border-top: 0;
 `,[D(`border`,{border:`var(--n-border-checked)`})])]),C(`disabled`,{cursor:`not-allowed`},[C(`checked`,[E(`checkbox-box`,`
 background-color: var(--n-color-disabled-checked);
 `,[D(`border`,{border:`var(--n-border-disabled-checked)`}),E(`checkbox-icon`,[f(`.check-icon, .line-icon`,{fill:`var(--n-check-mark-color-disabled-checked)`})])])]),E(`checkbox-box`,`
 background-color: var(--n-color-disabled);
 `,[D(`border`,`
 border: var(--n-border-disabled);
 `),E(`checkbox-icon`,[f(`.check-icon, .line-icon`,`
 fill: var(--n-check-mark-color-disabled);
 `)])]),D(`label`,`
 color: var(--n-text-color-disabled);
 `)]),E(`checkbox-box-wrapper`,`
 position: relative;
 width: var(--n-size);
 flex-shrink: 0;
 flex-grow: 0;
 user-select: none;
 -webkit-user-select: none;
 `),E(`checkbox-box`,`
 position: absolute;
 left: 0;
 top: 50%;
 transform: translateY(-50%);
 height: var(--n-size);
 width: var(--n-size);
 display: inline-block;
 box-sizing: border-box;
 border-radius: var(--n-border-radius);
 background-color: var(--n-color);
 transition: background-color 0.3s var(--n-bezier);
 `,[D(`border`,`
 transition:
 border-color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier);
 border-radius: inherit;
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 border: var(--n-border);
 `),E(`checkbox-icon`,`
 display: flex;
 align-items: center;
 justify-content: center;
 position: absolute;
 left: 1px;
 right: 1px;
 top: 1px;
 bottom: 1px;
 `,[f(`.check-icon, .line-icon`,`
 width: 100%;
 fill: var(--n-check-mark-color);
 opacity: 0;
 transform: scale(0.5);
 transform-origin: center;
 transition:
 fill 0.3s var(--n-bezier),
 transform 0.3s var(--n-bezier),
 opacity 0.3s var(--n-bezier),
 border-color 0.3s var(--n-bezier);
 `),i({left:`1px`,top:`1px`})])]),D(`label`,`
 color: var(--n-text-color);
 transition: color .3s var(--n-bezier);
 user-select: none;
 -webkit-user-select: none;
 padding: var(--n-label-padding);
 font-weight: var(--n-label-font-weight);
 `,[f(`&:empty`,{display:`none`})])]),U(E(`checkbox`,`
 --n-merged-color-table: var(--n-color-table-modal);
 `)),z(E(`checkbox`,`
 --n-merged-color-table: var(--n-color-table-popover);
 `))]),ut=W({name:`Checkbox`,props:Object.assign(Object.assign({},I.props),{size:String,checked:{type:[Boolean,String,Number],default:void 0},defaultChecked:{type:[Boolean,String,Number],default:!1},value:[String,Number],disabled:{type:Boolean,default:void 0},indeterminate:Boolean,label:String,focusable:{type:Boolean,default:!0},checkedValue:{type:[Boolean,String,Number],default:!0},uncheckedValue:{type:[Boolean,String,Number],default:!1},"onUpdate:checked":[Function,Array],onUpdateChecked:[Function,Array],privateInsideTable:Boolean,onChange:[Function,Array]}),setup(e){let t=v(at,null),n=B(null),{mergedClsPrefixRef:r,inlineThemeDisabled:i,mergedRtlRef:a,mergedComponentPropsRef:o}=H(e),c=B(e.defaultChecked),l=he(V(e,`checked`),c),u=s(()=>{if(t){let n=t.valueSetRef.value;return n&&e.value!==void 0?n.has(e.value):!1}else return l.value===e.checkedValue}),d=A(e,{mergedSize(n){let{size:r}=e;if(r!==void 0)return r;if(t){let{value:e}=t.mergedSizeRef;if(e!==void 0)return e}if(n){let{mergedSize:e}=n;if(e!==void 0)return e.value}return o?.value?.Checkbox?.size||`medium`},mergedDisabled(n){let{disabled:r}=e;if(r!==void 0)return r;if(t){if(t.disabledRef.value)return!0;let{maxRef:{value:e},checkedCountRef:n}=t;if(e!==void 0&&n.value>=e&&!u.value)return!0;let{minRef:{value:r}}=t;if(r!==void 0&&n.value<=r&&u.value)return!0}return n?n.disabled.value:!1}}),{mergedDisabledRef:f,mergedSizeRef:p}=d,m=I(`Checkbox`,`-checkbox`,lt,be,e,r);function h(n){if(t&&e.value!==void 0)t.toggleCheckbox(!u.value,e.value);else{let{onChange:t,"onUpdate:checked":r,onUpdateChecked:i}=e,{nTriggerFormInput:a,nTriggerFormChange:o}=d,s=u.value?e.uncheckedValue:e.checkedValue;r&&G(r,s,n),i&&G(i,s,n),t&&G(t,s,n),a(),o(),c.value=s}}function g(e){f.value||h(e)}function _(e){if(!f.value)switch(e.key){case` `:case`Enter`:h(e)}}function y(e){switch(e.key){case` `:e.preventDefault()}}let x={focus:()=>{var e;(e=n.value)==null||e.focus()},blur:()=>{var e;(e=n.value)==null||e.blur()}},S=b(`Checkbox`,a,r),C=P(()=>{let{value:e}=p,{common:{cubicBezierEaseInOut:t},self:{borderRadius:n,color:r,colorChecked:i,colorDisabled:a,colorTableHeader:o,colorTableHeaderModal:s,colorTableHeaderPopover:c,checkMarkColor:l,checkMarkColorDisabled:u,border:d,borderFocus:f,borderDisabled:h,borderChecked:g,boxShadowFocus:_,textColor:v,textColorDisabled:y,checkMarkColorDisabledChecked:b,colorDisabledChecked:x,borderDisabledChecked:S,labelPadding:C,labelLineHeight:w,labelFontWeight:T,[j(`fontSize`,e)]:E,[j(`size`,e)]:D}}=m.value;return{"--n-label-line-height":w,"--n-label-font-weight":T,"--n-size":D,"--n-bezier":t,"--n-border-radius":n,"--n-border":d,"--n-border-checked":g,"--n-border-focus":f,"--n-border-disabled":h,"--n-border-disabled-checked":S,"--n-box-shadow-focus":_,"--n-color":r,"--n-color-checked":i,"--n-color-table":o,"--n-color-table-modal":s,"--n-color-table-popover":c,"--n-color-disabled":a,"--n-color-disabled-checked":x,"--n-text-color":v,"--n-text-color-disabled":y,"--n-check-mark-color":l,"--n-check-mark-color-disabled":u,"--n-check-mark-color-disabled-checked":b,"--n-font-size":E,"--n-label-padding":C}}),w=i?N(`checkbox`,P(()=>p.value[0]),C,e):void 0;return Object.assign(d,x,{rtlEnabled:S,selfRef:n,mergedClsPrefix:r,mergedDisabled:f,renderedChecked:u,mergedTheme:m,labelId:Z(),handleClick:g,handleKeyUp:_,handleKeyDown:y,cssVars:i?void 0:C,themeClass:w?.themeClass,onRender:w?.onRender})},render(){var e;let{$slots:t,renderedChecked:n,mergedDisabled:r,indeterminate:i,privateInsideTable:o,cssVars:s,labelId:c,label:l,mergedClsPrefix:u,focusable:d,handleKeyUp:f,handleKeyDown:p,handleClick:h}=this;(e=this.onRender)==null||e.call(this);let g=K(t.default,e=>l||e?a(`span`,{class:`${u}-checkbox__label`,id:c},l||e):null);return a(`div`,{ref:`selfRef`,class:[`${u}-checkbox`,this.themeClass,this.rtlEnabled&&`${u}-checkbox--rtl`,n&&`${u}-checkbox--checked`,r&&`${u}-checkbox--disabled`,i&&`${u}-checkbox--indeterminate`,o&&`${u}-checkbox--inside-table`,g&&`${u}-checkbox--show-label`],tabindex:r||!d?void 0:0,role:`checkbox`,"aria-checked":i?`mixed`:n,"aria-labelledby":c,style:s,onKeyup:f,onKeydown:p,onClick:h,onMousedown:()=>{m(`selectstart`,window,e=>{e.preventDefault()},{once:!0})}},a(`div`,{class:`${u}-checkbox-box-wrapper`},`\xA0`,a(`div`,{class:`${u}-checkbox-box`},a(_,null,{default:()=>this.indeterminate?a(`div`,{key:`indeterminate`,class:`${u}-checkbox-icon`},ct()):a(`div`,{key:`check`,class:`${u}-checkbox-icon`},st())}),a(`div`,{class:`${u}-checkbox-box__border`}))),g)}}),dt=f([E(`select`,`
 z-index: auto;
 outline: none;
 width: 100%;
 position: relative;
 font-weight: var(--n-font-weight);
 `),E(`select-menu`,`
 margin: 4px 0;
 box-shadow: var(--n-menu-box-shadow);
 `,[Ee({originalTransition:`background-color .3s var(--n-bezier), box-shadow .3s var(--n-bezier)`})])]),ft=W({name:`Select`,props:Object.assign(Object.assign({},I.props),{to:ue.propTo,bordered:{type:Boolean,default:void 0},clearable:Boolean,clearCreatedOptionsOnClear:{type:Boolean,default:!0},clearFilterAfterSelect:{type:Boolean,default:!0},options:{type:Array,default:()=>[]},defaultValue:{type:[String,Number,Array],default:null},keyboard:{type:Boolean,default:!0},value:[String,Number,Array],placeholder:String,menuProps:Object,multiple:Boolean,size:String,menuSize:{type:String},filterable:Boolean,disabled:{type:Boolean,default:void 0},remote:Boolean,loading:Boolean,filter:Function,placement:{type:String,default:`bottom-start`},widthMode:{type:String,default:`trigger`},tag:Boolean,onCreate:Function,fallbackOption:{type:[Function,Boolean],default:void 0},show:{type:Boolean,default:void 0},showArrow:{type:Boolean,default:!0},maxTagCount:[Number,String],ellipsisTagPopoverProps:Object,consistentMenuWidth:{type:Boolean,default:!0},virtualScroll:{type:Boolean,default:!0},labelField:{type:String,default:`label`},valueField:{type:String,default:`value`},childrenField:{type:String,default:`children`},renderLabel:Function,renderOption:Function,renderTag:Function,"onUpdate:value":[Function,Array],inputProps:Object,nodeProps:Function,ignoreComposition:{type:Boolean,default:!0},showOnFocus:Boolean,onUpdateValue:[Function,Array],onBlur:[Function,Array],onClear:[Function,Array],onFocus:[Function,Array],onScroll:[Function,Array],onSearch:[Function,Array],onUpdateShow:[Function,Array],"onUpdate:show":[Function,Array],displayDirective:{type:String,default:`show`},resetMenuOnOptionsChange:{type:Boolean,default:!0},status:String,showCheckmark:{type:Boolean,default:!0},scrollbarProps:Object,onChange:[Function,Array],items:Array}),slots:Object,setup(e){let{mergedClsPrefixRef:t,mergedBorderedRef:n,namespaceRef:r,inlineThemeDisabled:i,mergedComponentPropsRef:a}=H(e),o=I(`Select`,`-select`,dt,Se,e,t),s=B(e.defaultValue),c=he(V(e,`value`),s),l=B(!1),u=B(``),d=ge(e,[`items`,`options`]),f=B([]),m=B([]),h=P(()=>m.value.concat(f.value).concat(d.value)),g=P(()=>{let{filter:t}=e;if(t)return t;let{labelField:n,valueField:r}=e;return(e,t)=>{if(!t)return!1;let i=t[n];if(typeof i==`string`)return tt(e,i);let a=t[r];return typeof a==`string`?tt(e,a):typeof a==`number`&&tt(e,String(a))}}),_=P(()=>{if(e.remote)return d.value;{let{value:t}=h,{value:n}=u;return!n.length||!e.filterable?t:rt(t,g.value,n,e.childrenField)}}),v=P(()=>{let{valueField:t,childrenField:n}=e,r=nt(t,n);return ae(_.value,r)}),y=P(()=>it(h.value,e.valueField,e.childrenField)),b=B(!1),x=he(V(e,`show`),b),C=B(null),T=B(null),E=B(null),{localeRef:D}=Ce(`Select`),ee=P(()=>e.placeholder??D.value.placeholder),O=[],k=B(new Map),j=P(()=>{let{fallbackOption:t}=e;if(t===void 0){let{labelField:t,valueField:n}=e;return e=>({[t]:String(e),[n]:e})}return t===!1?!1:e=>Object.assign(t(e),{value:e})});function M(t){let n=e.remote,{value:r}=k,{value:i}=y,{value:a}=j,o=[];return t.forEach(e=>{if(i.has(e))o.push(i.get(e));else if(n&&r.has(e))o.push(r.get(e));else if(a){let t=a(e);t&&o.push(t)}}),o}let F=P(()=>{if(e.multiple){let{value:e}=c;return Array.isArray(e)?M(e):[]}return null}),te=P(()=>{let{value:t}=c;return!e.multiple&&!Array.isArray(t)?t===null?null:M([t])[0]||null:null}),L=A(e,{mergedSize:t=>{let{size:n}=e;if(n)return n;let{mergedSize:r}=t||{};return r?.value?r.value:a?.value?.Select?.size||`medium`}}),{mergedSizeRef:R,mergedDisabledRef:z,mergedStatusRef:ne}=L;function U(t,n){let{onChange:r,"onUpdate:value":i,onUpdateValue:a}=e,{nTriggerFormChange:o,nTriggerFormInput:c}=L;r&&G(r,t,n),a&&G(a,t,n),i&&G(i,t,n),s.value=t,o(),c()}function W(t){let{onBlur:n}=e,{nTriggerFormBlur:r}=L;n&&G(n,t),r()}function K(){let{onClear:t}=e;t&&G(t)}function q(t){let{onFocus:n,showOnFocus:r}=e,{nTriggerFormFocus:i}=L;n&&G(n,t),i(),r&&Z()}function J(t){let{onSearch:n}=e;n&&G(n,t)}function Y(t){let{onScroll:n}=e;n&&G(n,t)}function ie(){var t;let{remote:n,multiple:r}=e;if(n){let{value:n}=k;if(r){let{valueField:r}=e;(t=F.value)==null||t.forEach(e=>{n.set(e[r],e)})}else{let t=te.value;t&&n.set(t[e.valueField],t)}}}function X(t){let{onUpdateShow:n,"onUpdate:show":r}=e;n&&G(n,t),r&&G(r,t),b.value=t}function Z(){z.value||(X(!0),b.value=!0,e.filterable&&Me())}function Q(){X(!1)}function oe(){u.value=``,m.value=O}let se=B(!1);function ce(){e.filterable&&(se.value=!0)}function le(){e.filterable&&(se.value=!1,x.value||oe())}function de(){z.value||(x.value?e.filterable?Me():Q():Z())}function fe(e){(E.value?.selfRef)?.contains(e.relatedTarget)||(l.value=!1,W(e),Q())}function pe(e){q(e),l.value=!0}function me(){l.value=!0}function _e(e){C.value?.$el.contains(e.relatedTarget)||(l.value=!1,W(e),Q())}function ye(){var e;(e=C.value)==null||e.focus(),Q()}function be(e){x.value&&(C.value?.$el.contains(p(e))||Q())}function xe(t){if(!Array.isArray(t))return[];if(j.value)return Array.from(t);{let{remote:n}=e,{value:r}=y;if(n){let{value:e}=k;return t.filter(t=>r.has(t)||e.has(t))}else return t.filter(e=>r.has(e))}}function we(e){Te(e.rawNode)}function Te(t){if(z.value)return;let{tag:n,remote:r,clearFilterAfterSelect:i,valueField:a}=e;if(n&&!r){let{value:e}=m,t=e[0]||null;if(t){let e=f.value;e.length?e.push(t):f.value=[t],m.value=O}}if(r&&k.value.set(t[a],t),e.multiple){let e=xe(c.value),o=e.findIndex(e=>e===t[a]);if(~o){if(e.splice(o,1),n&&!r){let e=Ee(t[a]);~e&&(f.value.splice(e,1),i&&(u.value=``))}}else e.push(t[a]),i&&(u.value=``);U(e,M(e))}else{if(n&&!r){let e=Ee(t[a]);~e?f.value=[f.value[e]]:f.value=O}je(),Q(),U(t[a],t)}}function Ee(t){return f.value.findIndex(n=>n[e.valueField]===t)}function $(t){x.value||Z();let{value:n}=t.target;u.value=n;let{tag:r,remote:i}=e;if(J(n),r&&!i){if(!n){m.value=O;return}let{onCreate:t}=e,r=t?t(n):{[e.labelField]:n,[e.valueField]:n},{valueField:i,labelField:a}=e;d.value.some(e=>e[i]===r[i]||e[a]===r[a])||f.value.some(e=>e[i]===r[i]||e[a]===r[a])?m.value=O:m.value=[r]}}function De(t){t.stopPropagation();let{multiple:n,tag:r,remote:i,clearCreatedOptionsOnClear:a}=e;!n&&e.filterable&&Q(),r&&!i&&a&&(f.value=O),K(),n?U([],[]):U(null,null)}function Oe(e){!re(e,`action`)&&!re(e,`empty`)&&!re(e,`header`)&&e.preventDefault()}function ke(e){Y(e)}function Ae(t){var n,r,i;if(!e.keyboard){t.preventDefault();return}switch(t.key){case` `:if(e.filterable)break;t.preventDefault();case`Enter`:if(!C.value?.isComposing){if(x.value){let t=E.value?.getPendingTmNode();t?we(t):e.filterable||(Q(),je())}else if(Z(),e.tag&&se.value){let t=m.value[0];if(t){let n=t[e.valueField],{value:r}=c;e.multiple&&Array.isArray(r)&&r.includes(n)||Te(t)}}}t.preventDefault();break;case`ArrowUp`:if(t.preventDefault(),e.loading)return;x.value&&((n=E.value)==null||n.prev());break;case`ArrowDown`:if(t.preventDefault(),e.loading)return;x.value?(r=E.value)==null||r.next():Z();break;case`Escape`:x.value&&(ve(t),Q()),(i=C.value)==null||i.focus();break}}function je(){var e;(e=C.value)==null||e.focus()}function Me(){var e;(e=C.value)==null||e.focusInput()}function Ne(){var e;x.value&&((e=T.value)==null||e.syncPosition())}ie(),w(V(e,`options`),ie);let Pe={focus:()=>{var e;(e=C.value)==null||e.focus()},focusInput:()=>{var e;(e=C.value)==null||e.focusInput()},blur:()=>{var e;(e=C.value)==null||e.blur()},blurInput:()=>{var e;(e=C.value)==null||e.blurInput()}},Fe=P(()=>{let{self:{menuBoxShadow:e}}=o.value;return{"--n-menu-box-shadow":e}}),Ie=i?N(`select`,void 0,Fe,e):void 0;return Object.assign(Object.assign({},Pe),{mergedStatus:ne,mergedClsPrefix:t,mergedBordered:n,namespace:r,treeMate:v,isMounted:S(),triggerRef:C,menuRef:E,pattern:u,uncontrolledShow:b,mergedShow:x,adjustedTo:ue(e),uncontrolledValue:s,mergedValue:c,followerRef:T,localizedPlaceholder:ee,selectedOption:te,selectedOptions:F,mergedSize:R,mergedDisabled:z,focused:l,activeWithoutMenuOpen:se,inlineThemeDisabled:i,onTriggerInputFocus:ce,onTriggerInputBlur:le,handleTriggerOrMenuResize:Ne,handleMenuFocus:me,handleMenuBlur:_e,handleMenuTabOut:ye,handleTriggerClick:de,handleToggle:we,handleDeleteOption:Te,handlePatternInput:$,handleClear:De,handleTriggerBlur:fe,handleTriggerFocus:pe,handleKeydown:Ae,handleMenuAfterLeave:oe,handleMenuClickOutside:be,handleMenuScroll:ke,handleMenuKeydown:Ae,handleMenuMousedown:Oe,mergedTheme:o,cssVars:i?void 0:Fe,themeClass:Ie?.themeClass,onRender:Ie?.onRender})},render(){return a(`div`,{class:`${this.mergedClsPrefix}-select`},a(se,null,{default:()=>[a(fe,null,{default:()=>a(Qe,{ref:`triggerRef`,inlineThemeDisabled:this.inlineThemeDisabled,status:this.mergedStatus,inputProps:this.inputProps,clsPrefix:this.mergedClsPrefix,showArrow:this.showArrow,maxTagCount:this.maxTagCount,ellipsisTagPopoverProps:this.ellipsisTagPopoverProps,bordered:this.mergedBordered,active:this.activeWithoutMenuOpen||this.mergedShow,pattern:this.pattern,placeholder:this.localizedPlaceholder,selectedOption:this.selectedOption,selectedOptions:this.selectedOptions,multiple:this.multiple,renderTag:this.renderTag,renderLabel:this.renderLabel,filterable:this.filterable,clearable:this.clearable,disabled:this.mergedDisabled,size:this.mergedSize,theme:this.mergedTheme.peers.InternalSelection,labelField:this.labelField,valueField:this.valueField,themeOverrides:this.mergedTheme.peerOverrides.InternalSelection,loading:this.loading,focused:this.focused,onClick:this.handleTriggerClick,onDeleteOption:this.handleDeleteOption,onPatternInput:this.handlePatternInput,onClear:this.handleClear,onBlur:this.handleTriggerBlur,onFocus:this.handleTriggerFocus,onKeydown:this.handleKeydown,onPatternBlur:this.onTriggerInputBlur,onPatternFocus:this.onTriggerInputFocus,onResize:this.handleTriggerOrMenuResize,ignoreComposition:this.ignoreComposition},{arrow:()=>{var e;return[(e=this.$slots).arrow?.call(e)]}})}),a(le,{ref:`followerRef`,show:this.mergedShow,to:this.adjustedTo,teleportDisabled:this.adjustedTo===ue.tdkey,containerClass:this.namespace,width:this.consistentMenuWidth?`target`:void 0,minWidth:`target`,placement:this.placement},{default:()=>a(O,{name:`fade-in-scale-up-transition`,appear:this.isMounted,onAfterLeave:this.handleMenuAfterLeave},{default:()=>{var e;return this.mergedShow||this.displayDirective===`show`?((e=this.onRender)==null||e.call(this),ne(a(Xe,Object.assign({},this.menuProps,{ref:`menuRef`,onResize:this.handleTriggerOrMenuResize,inlineThemeDisabled:this.inlineThemeDisabled,virtualScroll:this.consistentMenuWidth&&this.virtualScroll,class:[`${this.mergedClsPrefix}-select-menu`,this.themeClass,this.menuProps?.class],clsPrefix:this.mergedClsPrefix,focusable:!0,labelField:this.labelField,valueField:this.valueField,autoPending:!0,nodeProps:this.nodeProps,theme:this.mergedTheme.peers.InternalSelectMenu,themeOverrides:this.mergedTheme.peerOverrides.InternalSelectMenu,treeMate:this.treeMate,multiple:this.multiple,size:this.menuSize,renderOption:this.renderOption,renderLabel:this.renderLabel,value:this.mergedValue,style:[this.menuProps?.style,this.cssVars],onToggle:this.handleToggle,onScroll:this.handleMenuScroll,onFocus:this.handleMenuFocus,onBlur:this.handleMenuBlur,onKeydown:this.handleMenuKeydown,onTabOut:this.handleMenuTabOut,onMousedown:this.handleMenuMousedown,show:this.mergedShow,showCheckmark:this.showCheckmark,resetMenuOnOptionsChange:this.resetMenuOnOptionsChange,scrollbarProps:this.scrollbarProps}),{empty:()=>{var e;return[(e=this.$slots).empty?.call(e)]},header:()=>{var e;return[(e=this.$slots).header?.call(e)]},action:()=>{var e;return[(e=this.$slots).action?.call(e)]}}),this.displayDirective===`show`?[[k,this.mergedShow],[Q,this.handleMenuClickOutside,void 0,{capture:!0}]]:[[Q,this.handleMenuClickOutside,void 0,{capture:!0}]])):null}})})]}))}});export{Xe as a,Le as c,nt as i,ut as n,Ge as o,ot as r,Be as s,ft as t};