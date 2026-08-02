import{$t as e,A as t,An as n,Cn as r,Ct as i,Dt as a,En as o,Ft as s,Gn as c,Hn as l,In as u,Jn as d,Jt as f,Kn as p,Mt as m,N as h,Nt as g,On as _,Ot as v,P as y,Pn as b,Pt as x,Qt as S,Rn as C,S as w,Sn as T,T as E,Tt as D,Yt as O,Zt as k,_ as A,_n as j,an as M,bn as N,cn as ee,en as te,f as ne,fn as P,fr as F,ft as I,gn as L,i as re,j as R,jt as ie,k as ae,lr as z,nn as oe,pt as B,qn as V,rr as H,sr as U,tn as se,ur as ce,vn as W,wn as G,wt as K,x as le,xn as ue,xt as de,y as fe,yn as q,yt as pe,zn as J,zt as me}from"./client-CHTsrZM3.js";import{r as he,t as ge}from"./create-CXb85VLd.js";import{t as _e}from"./misc-DDs3MKLt.js";import{_ as ve,c as ye,f as be,g as xe,l as Se,o as Ce,p as we,r as Te,s as Ee}from"./light-FpSYQy6K.js";import{t as Y}from"./use-merged-state-DSdsnVdt.js";import{i as De,n as Oe,r as X,t as ke}from"./text-G-kXxcoP.js";import{i as Ae,n as je,r as Me,t as Ne}from"./useMobileViewport-BXGjzTie.js";import{r as Pe,t as Fe}from"./fade-in-height-expand.cssr-DPhGC2X4.js";import{r as Ie,t as Le}from"./Icon-DGOuQrf2.js";import{t as Re}from"./Alert-BMWoTUGt.js";import{A as Z,F as ze,M as Be,P as Ve,f as He,i as Ue,l as We,n as Ge,r as Ke,t as qe,w as Je}from"./index-L53TPgsP.js";import{t as Ye}from"./CodeSlashOutline-B7Iu-qav.js";import{t as Xe}from"./ReaderOutline-DJpgOlG0.js";import{t as Ze}from"./SwapHorizontalOutline-MTuhXynN.js";import{t as Qe}from"./PanelUpdateAction-C9Mod9Zp.js";import{t as $e}from"./backend-Q03GwhA2.js";var et=G({name:`ChevronDownFilled`,render(){return o(`svg`,{viewBox:`0 0 16 16`,fill:`none`,xmlns:`http://www.w3.org/2000/svg`},o(`path`,{d:`M3.20041 5.73966C3.48226 5.43613 3.95681 5.41856 4.26034 5.70041L8 9.22652L11.7397 5.70041C12.0432 5.41856 12.5177 5.43613 12.7996 5.73966C13.0815 6.0432 13.0639 6.51775 12.7603 6.7996L8.51034 10.7996C8.22258 11.0668 7.77743 11.0668 7.48967 10.7996L3.23966 6.7996C2.93613 6.51775 2.91856 6.0432 3.20041 5.73966Z`,fill:`currentColor`}))}}),tt=m&&`loading`in document.createElement(`img`);function nt(e={}){let{root:t=null}=e;return{hash:`${e.rootMargin||`0px 0px 0px 0px`}-${Array.isArray(e.threshold)?e.threshold.join(`,`):e.threshold??`0`}`,options:Object.assign(Object.assign({},e),{root:(typeof t==`string`?document.querySelector(t):t)||document.documentElement})}}var rt=new WeakMap,it=new WeakMap,at=new WeakMap,ot=(e,t,n)=>{if(!e)return()=>{};let r=nt(t),{root:i}=r.options,a,o=rt.get(i);o?a=o:(a=new Map,rt.set(i,a));let s,c;a.has(r.hash)?(c=a.get(r.hash),c[1].has(e)||(s=c[0],c[1].add(e),s.observe(e))):(s=new IntersectionObserver(e=>{e.forEach(e=>{if(e.isIntersecting){let t=it.get(e.target),n=at.get(e.target);t&&t(),n&&(n.value=!0)}})},r.options),s.observe(e),c=[s,new Set([e])],a.set(r.hash,c));let l=!1,u=()=>{l||(it.delete(e),at.delete(e),l=!0,c[1].has(e)&&(c[0].unobserve(e),c[1].delete(e)),c[1].size<=0&&a.delete(r.hash),a.size||rt.delete(i))};return it.set(e,u),at.set(e,n),u},st=g(`n-avatar-group`),ct=O(`avatar`,`
 width: var(--n-merged-size);
 height: var(--n-merged-size);
 color: #FFF;
 font-size: var(--n-font-size);
 display: inline-flex;
 position: relative;
 overflow: hidden;
 text-align: center;
 border: var(--n-border);
 border-radius: var(--n-border-radius);
 --n-merged-color: var(--n-color);
 background-color: var(--n-merged-color);
 transition:
 border-color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
`,[se(f(`&`,`--n-merged-color: var(--n-color-modal);`)),oe(f(`&`,`--n-merged-color: var(--n-color-popover);`)),f(`img`,`
 width: 100%;
 height: 100%;
 `),k(`text`,`
 white-space: nowrap;
 display: inline-block;
 position: absolute;
 left: 50%;
 top: 50%;
 `),O(`icon`,`
 vertical-align: bottom;
 font-size: calc(var(--n-merged-size) - 6px);
 `),k(`text`,`line-height: 1.25`)]),lt=G({name:`Avatar`,props:Object.assign(Object.assign({},R.props),{size:[String,Number],src:String,circle:{type:Boolean,default:void 0},objectFit:String,round:{type:Boolean,default:void 0},bordered:{type:Boolean,default:void 0},onError:Function,fallbackSrc:String,intersectionObserverOptions:Object,lazy:Boolean,onLoad:Function,renderPlaceholder:Function,renderFallback:Function,imgProps:Object,color:String}),slots:Object,setup(e){let{mergedClsPrefixRef:t,inlineThemeDisabled:n}=B(e),r=H(!1),i=null,o=H(null),s=H(null),l=()=>{let{value:e}=o;if(e&&(i===null||i!==e.innerHTML)){i=e.innerHTML;let{value:t}=s;if(t){let{offsetWidth:n,offsetHeight:r}=t,{offsetWidth:i,offsetHeight:a}=e,o=.9,s=Math.min(n/i*o,r/a*o,1);e.style.transform=`translateX(-50%) translateY(-50%) scale(${s})`}}},d=_(st,null),f=L(()=>{let{size:t}=e;if(t)return t;let{size:n}=d||{};return n||`medium`}),m=R(`Avatar`,`-avatar`,ct,Je,e,t),h=_(Ie,null),g=L(()=>{if(d)return!0;let{round:t,circle:n}=e;return t!==void 0||n!==void 0?t||n:h?h.roundRef.value:!1}),v=L(()=>d?!0:e.bordered||!1),y=L(()=>{let t=f.value,n=g.value,r=v.value,{color:i}=e,{self:{borderRadius:a,fontSize:o,color:s,border:c,colorModal:l,colorPopover:u},common:{cubicBezierEaseInOut:d}}=m.value,p;return p=typeof t==`number`?`${t}px`:m.value.self[te(`height`,t)],{"--n-font-size":o,"--n-border":r?c:`none`,"--n-border-radius":n?`50%`:a,"--n-color":i||s,"--n-color-modal":i||l,"--n-color-popover":i||u,"--n-bezier":d,"--n-merged-size":`var(--n-avatar-size-override, ${p})`}}),x=n?I(`avatar`,L(()=>{let t=f.value,n=g.value,r=v.value,{color:i}=e,o=``;return t&&(typeof t==`number`?o+=`a${t}`:o+=t[0]),n&&(o+=`b`),r&&(o+=`c`),i&&(o+=a(i)),o}),y,e):void 0,S=H(!e.lazy);u(()=>{if(e.lazy&&e.intersectionObserverOptions){let t,n=p(()=>{t?.(),t=void 0,e.lazy&&(t=ot(s.value,e.intersectionObserverOptions,S))});b(()=>{n(),t?.()})}}),c(()=>e.src||e.imgProps?.src,()=>{r.value=!1});let C=H(!e.lazy);return{textRef:o,selfRef:s,mergedRoundRef:g,mergedClsPrefix:t,fitTextTransform:l,cssVars:n?void 0:y,themeClass:x?.themeClass,onRender:x?.onRender,hasLoadError:r,shouldStartLoading:S,loaded:C,mergedOnError:t=>{if(!S.value)return;r.value=!0;let{onError:n,imgProps:{onError:i}={}}=e;n?.(t),i?.(t)},mergedOnLoad:t=>{let{onLoad:n,imgProps:{onLoad:r}={}}=e;n?.(t),r?.(t),C.value=!0}}},render(){var e;let{$slots:t,src:n,mergedClsPrefix:r,lazy:i,onRender:a,loaded:s,hasLoadError:c,imgProps:l={}}=this;a?.();let u,d=!s&&!c&&(this.renderPlaceholder?this.renderPlaceholder():(e=this.$slots).placeholder?.call(e));return u=this.hasLoadError?this.renderFallback?this.renderFallback():pe(t.fallback,()=>[o(`img`,{src:this.fallbackSrc,style:{objectFit:this.objectFit}})]):de(t.default,e=>{if(e)return o(v,{onResize:this.fitTextTransform},{default:()=>o(`span`,{ref:`textRef`,class:`${r}-avatar__text`},e)});if(n||l.src){let e=this.src||l.src;return o(`img`,Object.assign(Object.assign({},l),{loading:tt&&!this.intersectionObserverOptions&&i?`lazy`:`eager`,src:i&&this.intersectionObserverOptions?this.shouldStartLoading?e:void 0:e,"data-image-src":e,onLoad:this.mergedOnLoad,onError:this.mergedOnError,style:[l.style||``,{objectFit:this.objectFit},d?{height:`0`,width:`0`,visibility:`hidden`,position:`absolute`}:``]}))}}),o(`span`,{ref:`selfRef`,class:[`${r}-avatar`,this.themeClass],style:this.cssVars},u,i&&d)}}),ut=G({name:`NDrawerContent`,inheritAttrs:!1,props:{blockScroll:Boolean,show:{type:Boolean,default:void 0},displayDirective:{type:String,required:!0},placement:{type:String,required:!0},contentClass:String,contentStyle:[Object,String],nativeScrollbar:{type:Boolean,required:!0},scrollbarProps:Object,trapFocus:{type:Boolean,default:!0},autoFocus:{type:Boolean,default:!0},showMask:{type:[Boolean,String],required:!0},maxWidth:Number,maxHeight:Number,minWidth:Number,minHeight:Number,resizable:Boolean,onClickoutside:Function,onAfterLeave:Function,onAfterEnter:Function,onEsc:Function},setup(e){let t=H(!!e.show),n=H(null),r=_(ve),i=0,a=``,o=null,s=H(!1),l=H(!1),u=L(()=>e.placement===`top`||e.placement===`bottom`),{mergedClsPrefixRef:d,mergedRtlRef:f}=B(e),m=y(`Drawer`,f,d),h=D,g=e=>{l.value=!0,i=u.value?e.clientY:e.clientX,a=document.body.style.cursor,document.body.style.cursor=u.value?`ns-resize`:`ew-resize`,document.body.addEventListener(`mousemove`,E),document.body.addEventListener(`mouseleave`,h),document.body.addEventListener(`mouseup`,D)},v=()=>{o!==null&&(window.clearTimeout(o),o=null),l.value?s.value=!0:o=window.setTimeout(()=>{s.value=!0},300)},x=()=>{o!==null&&(window.clearTimeout(o),o=null),s.value=!1},{doUpdateHeight:S,doUpdateWidth:C}=r,w=t=>{let{maxWidth:n}=e;if(n&&t>n)return n;let{minWidth:r}=e;return r&&t<r?r:t},T=t=>{let{maxHeight:n}=e;if(n&&t>n)return n;let{minHeight:r}=e;return r&&t<r?r:t};function E(t){if(l.value)if(u.value){let r=n.value?.offsetHeight||0,a=i-t.clientY;r+=e.placement===`bottom`?a:-a,r=T(r),S(r),i=t.clientY}else{let r=n.value?.offsetWidth||0,a=i-t.clientX;r+=e.placement===`right`?a:-a,r=w(r),C(r),i=t.clientX}}function D(){l.value&&(i=0,l.value=!1,document.body.style.cursor=a,document.body.removeEventListener(`mousemove`,E),document.body.removeEventListener(`mouseup`,D),document.body.removeEventListener(`mouseleave`,h))}p(()=>{e.show&&(t.value=!0)}),c(()=>e.show,e=>{e||D()}),b(()=>{D()});let O=L(()=>{let{show:t}=e,n=[[ee,t]];return e.showMask||n.push([Se,e.onClickoutside,void 0,{capture:!0}]),n});function k(){var n;t.value=!1,(n=e.onAfterLeave)==null||n.call(e)}return Ve(L(()=>e.blockScroll&&t.value)),J(xe,n),J(be,null),J(we,null),{bodyRef:n,rtlEnabled:m,mergedClsPrefix:r.mergedClsPrefixRef,isMounted:r.isMountedRef,mergedTheme:r.mergedThemeRef,displayed:t,transitionName:L(()=>({right:`slide-in-from-right-transition`,left:`slide-in-from-left-transition`,top:`slide-in-from-top-transition`,bottom:`slide-in-from-bottom-transition`})[e.placement]),handleAfterLeave:k,bodyDirectives:O,handleMousedownResizeTrigger:g,handleMouseenterResizeTrigger:v,handleMouseleaveResizeTrigger:x,isDragging:l,isHoverOnResizeTrigger:s}},render(){let{$slots:e,mergedClsPrefix:t}=this;return this.displayDirective===`show`||this.displayed||this.show?d(o(`div`,{role:`none`},o(Ce,{disabled:!this.showMask||!this.trapFocus,active:this.show,autoFocus:this.autoFocus,onEsc:this.onEsc},{default:()=>o(M,{name:this.transitionName,appear:this.isMounted,onAfterEnter:this.onAfterEnter,onAfterLeave:this.handleAfterLeave},{default:()=>d(o(`div`,n(this.$attrs,{role:`dialog`,ref:`bodyRef`,"aria-modal":`true`,class:[`${t}-drawer`,this.rtlEnabled&&`${t}-drawer--rtl`,`${t}-drawer--${this.placement}-placement`,this.isDragging&&`${t}-drawer--unselectable`,this.nativeScrollbar&&`${t}-drawer--native-scrollbar`]}),[this.resizable?o(`div`,{class:[`${t}-drawer__resize-trigger`,(this.isDragging||this.isHoverOnResizeTrigger)&&`${t}-drawer__resize-trigger--hover`],onMouseenter:this.handleMouseenterResizeTrigger,onMouseleave:this.handleMouseleaveResizeTrigger,onMousedown:this.handleMousedownResizeTrigger}):null,this.nativeScrollbar?o(`div`,{class:[`${t}-drawer-content-wrapper`,this.contentClass],style:this.contentStyle,role:`none`},e):o(A,Object.assign({},this.scrollbarProps,{contentStyle:this.contentStyle,contentClass:[`${t}-drawer-content-wrapper`,this.contentClass],theme:this.mergedTheme.peers.Scrollbar,themeOverrides:this.mergedTheme.peerOverrides.Scrollbar}),e)]),this.bodyDirectives)})})),[[ee,this.displayDirective===`if`||this.displayed||this.show]]):null}}),{cubicBezierEaseIn:dt,cubicBezierEaseOut:ft}=h;function pt({duration:e=`0.3s`,leaveDuration:t=`0.2s`,name:n=`slide-in-from-bottom`}={}){return[f(`&.${n}-transition-leave-active`,{transition:`transform ${t} ${dt}`}),f(`&.${n}-transition-enter-active`,{transition:`transform ${e} ${ft}`}),f(`&.${n}-transition-enter-to`,{transform:`translateY(0)`}),f(`&.${n}-transition-enter-from`,{transform:`translateY(100%)`}),f(`&.${n}-transition-leave-from`,{transform:`translateY(0)`}),f(`&.${n}-transition-leave-to`,{transform:`translateY(100%)`})]}var{cubicBezierEaseIn:mt,cubicBezierEaseOut:ht}=h;function gt({duration:e=`0.3s`,leaveDuration:t=`0.2s`,name:n=`slide-in-from-left`}={}){return[f(`&.${n}-transition-leave-active`,{transition:`transform ${t} ${mt}`}),f(`&.${n}-transition-enter-active`,{transition:`transform ${e} ${ht}`}),f(`&.${n}-transition-enter-to`,{transform:`translateX(0)`}),f(`&.${n}-transition-enter-from`,{transform:`translateX(-100%)`}),f(`&.${n}-transition-leave-from`,{transform:`translateX(0)`}),f(`&.${n}-transition-leave-to`,{transform:`translateX(-100%)`})]}var{cubicBezierEaseIn:_t,cubicBezierEaseOut:vt}=h;function yt({duration:e=`0.3s`,leaveDuration:t=`0.2s`,name:n=`slide-in-from-right`}={}){return[f(`&.${n}-transition-leave-active`,{transition:`transform ${t} ${_t}`}),f(`&.${n}-transition-enter-active`,{transition:`transform ${e} ${vt}`}),f(`&.${n}-transition-enter-to`,{transform:`translateX(0)`}),f(`&.${n}-transition-enter-from`,{transform:`translateX(100%)`}),f(`&.${n}-transition-leave-from`,{transform:`translateX(0)`}),f(`&.${n}-transition-leave-to`,{transform:`translateX(100%)`})]}var{cubicBezierEaseIn:bt,cubicBezierEaseOut:xt}=h;function St({duration:e=`0.3s`,leaveDuration:t=`0.2s`,name:n=`slide-in-from-top`}={}){return[f(`&.${n}-transition-leave-active`,{transition:`transform ${t} ${bt}`}),f(`&.${n}-transition-enter-active`,{transition:`transform ${e} ${xt}`}),f(`&.${n}-transition-enter-to`,{transform:`translateY(0)`}),f(`&.${n}-transition-enter-from`,{transform:`translateY(-100%)`}),f(`&.${n}-transition-leave-from`,{transform:`translateY(0)`}),f(`&.${n}-transition-leave-to`,{transform:`translateY(-100%)`})]}var Ct=f([O(`drawer`,`
 word-break: break-word;
 line-height: var(--n-line-height);
 position: absolute;
 pointer-events: all;
 box-shadow: var(--n-box-shadow);
 transition:
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 background-color: var(--n-color);
 color: var(--n-text-color);
 box-sizing: border-box;
 `,[yt(),gt(),St(),pt(),S(`unselectable`,`
 user-select: none; 
 -webkit-user-select: none;
 `),S(`native-scrollbar`,[O(`drawer-content-wrapper`,`
 overflow: auto;
 height: 100%;
 `)]),k(`resize-trigger`,`
 position: absolute;
 background-color: #0000;
 transition: background-color .3s var(--n-bezier);
 `,[S(`hover`,`
 background-color: var(--n-resize-trigger-color-hover);
 `)]),O(`drawer-content-wrapper`,`
 box-sizing: border-box;
 `),O(`drawer-content`,`
 height: 100%;
 display: flex;
 flex-direction: column;
 `,[S(`native-scrollbar`,[O(`drawer-body-content-wrapper`,`
 height: 100%;
 overflow: auto;
 `)]),O(`drawer-body`,`
 flex: 1 0 0;
 overflow: hidden;
 `),O(`drawer-body-content-wrapper`,`
 box-sizing: border-box;
 padding: var(--n-body-padding);
 `),O(`drawer-header`,`
 font-weight: var(--n-title-font-weight);
 line-height: 1;
 font-size: var(--n-title-font-size);
 color: var(--n-title-text-color);
 padding: var(--n-header-padding);
 transition: border .3s var(--n-bezier);
 border-bottom: 1px solid var(--n-divider-color);
 border-bottom: var(--n-header-border-bottom);
 display: flex;
 justify-content: space-between;
 align-items: center;
 `,[k(`main`,`
 flex: 1;
 `),k(`close`,`
 margin-left: 6px;
 transition:
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 `)]),O(`drawer-footer`,`
 display: flex;
 justify-content: flex-end;
 border-top: var(--n-footer-border-top);
 transition: border .3s var(--n-bezier);
 padding: var(--n-footer-padding);
 `)]),S(`right-placement`,`
 top: 0;
 bottom: 0;
 right: 0;
 border-top-left-radius: var(--n-border-radius);
 border-bottom-left-radius: var(--n-border-radius);
 `,[k(`resize-trigger`,`
 width: 3px;
 height: 100%;
 top: 0;
 left: 0;
 transform: translateX(-1.5px);
 cursor: ew-resize;
 `)]),S(`left-placement`,`
 top: 0;
 bottom: 0;
 left: 0;
 border-top-right-radius: var(--n-border-radius);
 border-bottom-right-radius: var(--n-border-radius);
 `,[k(`resize-trigger`,`
 width: 3px;
 height: 100%;
 top: 0;
 right: 0;
 transform: translateX(1.5px);
 cursor: ew-resize;
 `)]),S(`top-placement`,`
 top: 0;
 left: 0;
 right: 0;
 border-bottom-left-radius: var(--n-border-radius);
 border-bottom-right-radius: var(--n-border-radius);
 `,[k(`resize-trigger`,`
 width: 100%;
 height: 3px;
 bottom: 0;
 left: 0;
 transform: translateY(1.5px);
 cursor: ns-resize;
 `)]),S(`bottom-placement`,`
 left: 0;
 bottom: 0;
 right: 0;
 border-top-left-radius: var(--n-border-radius);
 border-top-right-radius: var(--n-border-radius);
 `,[k(`resize-trigger`,`
 width: 100%;
 height: 3px;
 top: 0;
 left: 0;
 transform: translateY(-1.5px);
 cursor: ns-resize;
 `)])]),f(`body`,[f(`>`,[O(`drawer-container`,`
 position: fixed;
 `)])]),O(`drawer-container`,`
 position: relative;
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 pointer-events: none;
 `,[f(`> *`,`
 pointer-events: all;
 `)]),O(`drawer-mask`,`
 background-color: rgba(0, 0, 0, .3);
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 `,[S(`invisible`,`
 background-color: rgba(0, 0, 0, 0)
 `),w({enterDuration:`0.2s`,leaveDuration:`0.2s`,enterCubicBezier:`var(--n-bezier-in)`,leaveCubicBezier:`var(--n-bezier-out)`})])]),wt=G({name:`Drawer`,inheritAttrs:!1,props:Object.assign(Object.assign({},R.props),{show:Boolean,width:[Number,String],height:[Number,String],placement:{type:String,default:`right`},maskClosable:{type:Boolean,default:!0},showMask:{type:[Boolean,String],default:!0},to:[String,Object],displayDirective:{type:String,default:`if`},nativeScrollbar:{type:Boolean,default:!0},zIndex:Number,onMaskClick:Function,scrollbarProps:Object,contentClass:String,contentStyle:[Object,String],trapFocus:{type:Boolean,default:!0},onEsc:Function,autoFocus:{type:Boolean,default:!0},closeOnEsc:{type:Boolean,default:!0},blockScroll:{type:Boolean,default:!0},maxWidth:Number,maxHeight:Number,minWidth:Number,minHeight:Number,resizable:Boolean,defaultWidth:{type:[Number,String],default:251},defaultHeight:{type:[Number,String],default:251},onUpdateWidth:[Function,Array],onUpdateHeight:[Function,Array],"onUpdate:width":[Function,Array],"onUpdate:height":[Function,Array],"onUpdate:show":[Function,Array],onUpdateShow:[Function,Array],onAfterEnter:Function,onAfterLeave:Function,drawerStyle:[String,Object],drawerClass:String,target:null,onShow:Function,onHide:Function}),setup(e){let{mergedClsPrefixRef:t,namespaceRef:n,inlineThemeDisabled:r}=B(e),i=x(),a=R(`Drawer`,`-drawer`,Ct,He,e,t),o=H(e.defaultWidth),s=H(e.defaultHeight),c=Y(U(e,`width`),o),l=Y(U(e,`height`),s),u=L(()=>{let{placement:t}=e;return t===`top`||t===`bottom`?``:X(c.value)}),d=L(()=>{let{placement:t}=e;return t===`left`||t===`right`?``:X(l.value)}),f=t=>{let{onUpdateWidth:n,"onUpdate:width":r}=e;n&&K(n,t),r&&K(r,t),o.value=t},p=t=>{let{onUpdateHeight:n,"onUpdate:width":r}=e;n&&K(n,t),r&&K(r,t),s.value=t},m=L(()=>[{width:u.value,height:d.value},e.drawerStyle||``]);function h(t){let{onMaskClick:n,maskClosable:r}=e;r&&y(!1),n&&n(t)}function g(e){h(e)}let _=ze();function v(t){var n;(n=e.onEsc)==null||n.call(e),e.show&&e.closeOnEsc&&Be(t)&&(_.value||y(!1))}function y(t){let{onHide:n,onUpdateShow:r,"onUpdate:show":i}=e;r&&K(r,t),i&&K(i,t),n&&!t&&K(n,t)}J(ve,{isMountedRef:i,mergedThemeRef:a,mergedClsPrefixRef:t,doUpdateShow:y,doUpdateHeight:p,doUpdateWidth:f});let b=L(()=>{let{common:{cubicBezierEaseInOut:e,cubicBezierEaseIn:t,cubicBezierEaseOut:n},self:{color:r,textColor:i,boxShadow:o,lineHeight:s,headerPadding:c,footerPadding:l,borderRadius:u,bodyPadding:d,titleFontSize:f,titleTextColor:p,titleFontWeight:m,headerBorderBottom:h,footerBorderTop:g,closeIconColor:_,closeIconColorHover:v,closeIconColorPressed:y,closeColorHover:b,closeColorPressed:x,closeIconSize:S,closeSize:C,closeBorderRadius:w,resizableTriggerColorHover:T}}=a.value;return{"--n-line-height":s,"--n-color":r,"--n-border-radius":u,"--n-text-color":i,"--n-box-shadow":o,"--n-bezier":e,"--n-bezier-out":n,"--n-bezier-in":t,"--n-header-padding":c,"--n-body-padding":d,"--n-footer-padding":l,"--n-title-text-color":p,"--n-title-font-size":f,"--n-title-font-weight":m,"--n-header-border-bottom":h,"--n-footer-border-top":g,"--n-close-icon-color":_,"--n-close-icon-color-hover":v,"--n-close-icon-color-pressed":y,"--n-close-size":C,"--n-close-color-hover":b,"--n-close-color-pressed":x,"--n-close-icon-size":S,"--n-close-border-radius":w,"--n-resize-trigger-color-hover":T}}),S=r?I(`drawer`,void 0,b,e):void 0;return{mergedClsPrefix:t,namespace:n,mergedBodyStyle:m,handleOutsideClick:g,handleMaskClick:h,handleEsc:v,mergedTheme:a,cssVars:r?void 0:b,themeClass:S?.themeClass,onRender:S?.onRender,isMounted:i}},render(){let{mergedClsPrefix:e}=this;return o(Ee,{to:this.to,show:this.show},{default:()=>{var t;return(t=this.onRender)==null||t.call(this),d(o(`div`,{class:[`${e}-drawer-container`,this.namespace,this.themeClass],style:this.cssVars,role:`none`},this.showMask?o(M,{name:`fade-in-transition`,appear:this.isMounted},{default:()=>this.show?o(`div`,{"aria-hidden":!0,class:[`${e}-drawer-mask`,this.showMask===`transparent`&&`${e}-drawer-mask--invisible`],onClick:this.handleMaskClick}):null}):null,o(ut,Object.assign({},this.$attrs,{class:[this.drawerClass,this.$attrs.class],style:[this.mergedBodyStyle,this.$attrs.style],blockScroll:this.blockScroll,contentStyle:this.contentStyle,contentClass:this.contentClass,placement:this.placement,scrollbarProps:this.scrollbarProps,show:this.show,displayDirective:this.displayDirective,nativeScrollbar:this.nativeScrollbar,onAfterEnter:this.onAfterEnter,onAfterLeave:this.onAfterLeave,trapFocus:this.trapFocus,autoFocus:this.autoFocus,resizable:this.resizable,maxHeight:this.maxHeight,minHeight:this.minHeight,maxWidth:this.maxWidth,minWidth:this.minWidth,showMask:this.showMask,onEsc:this.handleEsc,onClickoutside:this.handleOutsideClick}),this.$slots)),[[ye,{zIndex:this.zIndex,enabled:this.show}]])}})}}),Tt=G({name:`DrawerContent`,props:{title:String,headerClass:String,headerStyle:[Object,String],footerClass:String,footerStyle:[Object,String],bodyClass:String,bodyStyle:[Object,String],bodyContentClass:String,bodyContentStyle:[Object,String],nativeScrollbar:{type:Boolean,default:!0},scrollbarProps:Object,closable:Boolean},slots:Object,setup(){let e=_(ve,null);e||D(`drawer-content`,"`n-drawer-content` must be placed inside `n-drawer`.");let{doUpdateShow:t}=e;function n(){t(!1)}return{handleCloseClick:n,mergedTheme:e.mergedThemeRef,mergedClsPrefix:e.mergedClsPrefixRef}},render(){let{title:e,mergedClsPrefix:t,nativeScrollbar:n,mergedTheme:r,bodyClass:i,bodyStyle:a,bodyContentClass:s,bodyContentStyle:c,headerClass:l,headerStyle:u,footerClass:d,footerStyle:f,scrollbarProps:p,closable:m,$slots:h}=this;return o(`div`,{role:`none`,class:[`${t}-drawer-content`,n&&`${t}-drawer-content--native-scrollbar`]},h.header||e||m?o(`div`,{class:[`${t}-drawer-header`,l],style:u,role:`none`},o(`div`,{class:`${t}-drawer-header__main`,role:`heading`,"aria-level":`1`},h.header===void 0?e:h.header()),m&&o(Pe,{onClick:this.handleCloseClick,clsPrefix:t,class:`${t}-drawer-header__close`,absolute:!0})):null,n?o(`div`,{class:[`${t}-drawer-body`,i],style:a,role:`none`},o(`div`,{class:[`${t}-drawer-body-content-wrapper`,s],style:c,role:`none`},h)):o(A,Object.assign({themeOverrides:r.peerOverrides.Scrollbar,theme:r.peers.Scrollbar},p,{class:`${t}-drawer-body`,contentClass:[`${t}-drawer-body-content-wrapper`,s],contentStyle:c}),h),h.footer?o(`div`,{class:[`${t}-drawer-footer`,d],style:f,role:`none`},h.footer()):null)}});function Et(e){let{baseColor:t,textColor2:n,bodyColor:r,cardColor:i,dividerColor:a,actionColor:o,scrollbarColor:s,scrollbarColorHover:c,invertedColor:l}=e;return{textColor:n,textColorInverted:`#FFF`,color:r,colorEmbedded:o,headerColor:i,headerColorInverted:l,footerColor:o,footerColorInverted:l,headerBorderColor:a,headerBorderColorInverted:l,footerBorderColor:a,footerBorderColorInverted:l,siderBorderColor:a,siderBorderColorInverted:l,siderColor:i,siderColorInverted:l,siderToggleButtonBorder:`1px solid ${a}`,siderToggleButtonColor:t,siderToggleButtonIconColor:n,siderToggleButtonIconColorInverted:n,siderToggleBarColor:me(r,s),siderToggleBarColorHover:me(r,c),__invertScrollbar:`true`}}var Dt=t({name:`Layout`,common:le,peers:{Scrollbar:fe},self:Et}),Ot=g(`n-layout-sider`),kt={type:String,default:`static`},At=O(`layout`,`
 color: var(--n-text-color);
 background-color: var(--n-color);
 box-sizing: border-box;
 position: relative;
 z-index: auto;
 flex: auto;
 overflow: hidden;
 transition:
 box-shadow .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
`,[O(`layout-scroll-container`,`
 overflow-x: hidden;
 box-sizing: border-box;
 height: 100%;
 `),S(`absolute-positioned`,`
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 `)]),jt={embedded:Boolean,position:kt,nativeScrollbar:{type:Boolean,default:!0},scrollbarProps:Object,onScroll:Function,contentClass:String,contentStyle:{type:[String,Object],default:``},hasSider:Boolean,siderPlacement:{type:String,default:`left`}},Mt=g(`n-layout`);function Nt(e){return G({name:e?`LayoutContent`:`Layout`,props:Object.assign(Object.assign({},R.props),jt),setup(e){let t=H(null),n=H(null),{mergedClsPrefixRef:r,inlineThemeDisabled:i}=B(e),a=R(`Layout`,`-layout`,At,Dt,e,r);function o(r,i){if(e.nativeScrollbar){let{value:e}=t;e&&(i===void 0?e.scrollTo(r):e.scrollTo(r,i))}else{let{value:e}=n;e&&e.scrollTo(r,i)}}J(Mt,e);let s=0,c=0,l=t=>{var n;let r=t.target;s=r.scrollLeft,c=r.scrollTop,(n=e.onScroll)==null||n.call(e,t)};ie(()=>{if(e.nativeScrollbar){let e=t.value;e&&(e.scrollTop=c,e.scrollLeft=s)}});let u={display:`flex`,flexWrap:`nowrap`,width:`100%`,flexDirection:`row`},d={scrollTo:o},f=L(()=>{let{common:{cubicBezierEaseInOut:t},self:n}=a.value;return{"--n-bezier":t,"--n-color":e.embedded?n.colorEmbedded:n.color,"--n-text-color":n.textColor}}),p=i?I(`layout`,L(()=>e.embedded?`e`:``),f,e):void 0;return Object.assign({mergedClsPrefix:r,scrollableElRef:t,scrollbarInstRef:n,hasSiderStyle:u,mergedTheme:a,handleNativeElScroll:l,cssVars:i?void 0:f,themeClass:p?.themeClass,onRender:p?.onRender},d)},render(){var t;let{mergedClsPrefix:n,hasSider:r}=this;(t=this.onRender)==null||t.call(this);let i=r?this.hasSiderStyle:void 0;return o(`div`,{class:[this.themeClass,e&&`${n}-layout-content`,`${n}-layout`,`${n}-layout--${this.position}-positioned`],style:this.cssVars},this.nativeScrollbar?o(`div`,{ref:`scrollableElRef`,class:[`${n}-layout-scroll-container`,this.contentClass],style:[this.contentStyle,i],onScroll:this.handleNativeElScroll},this.$slots):o(A,Object.assign({},this.scrollbarProps,{onScroll:this.onScroll,ref:`scrollbarInstRef`,theme:this.mergedTheme.peers.Scrollbar,themeOverrides:this.mergedTheme.peerOverrides.Scrollbar,contentClass:this.contentClass,contentStyle:[this.contentStyle,i]}),this.$slots))}})}var Pt=Nt(!1),Ft=Nt(!0),It=O(`layout-header`,`
 transition:
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 box-sizing: border-box;
 width: 100%;
 background-color: var(--n-color);
 color: var(--n-text-color);
`,[S(`absolute-positioned`,`
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 `),S(`bordered`,`
 border-bottom: solid 1px var(--n-border-color);
 `)]),Lt={position:kt,inverted:Boolean,bordered:{type:Boolean,default:!1}},Rt=G({name:`LayoutHeader`,props:Object.assign(Object.assign({},R.props),Lt),setup(e){let{mergedClsPrefixRef:t,inlineThemeDisabled:n}=B(e),r=R(`Layout`,`-layout-header`,It,Dt,e,t),i=L(()=>{let{common:{cubicBezierEaseInOut:t},self:n}=r.value,i={"--n-bezier":t};return e.inverted?(i[`--n-color`]=n.headerColorInverted,i[`--n-text-color`]=n.textColorInverted,i[`--n-border-color`]=n.headerBorderColorInverted):(i[`--n-color`]=n.headerColor,i[`--n-text-color`]=n.textColor,i[`--n-border-color`]=n.headerBorderColor),i}),a=n?I(`layout-header`,L(()=>e.inverted?`a`:`b`),i,e):void 0;return{mergedClsPrefix:t,cssVars:n?void 0:i,themeClass:a?.themeClass,onRender:a?.onRender}},render(){var e;let{mergedClsPrefix:t}=this;return(e=this.onRender)==null||e.call(this),o(`div`,{class:[`${t}-layout-header`,this.themeClass,this.position&&`${t}-layout-header--${this.position}-positioned`,this.bordered&&`${t}-layout-header--bordered`],style:this.cssVars},this.$slots)}}),zt=O(`layout-sider`,`
 flex-shrink: 0;
 box-sizing: border-box;
 position: relative;
 z-index: 1;
 color: var(--n-text-color);
 transition:
 color .3s var(--n-bezier),
 border-color .3s var(--n-bezier),
 min-width .3s var(--n-bezier),
 max-width .3s var(--n-bezier),
 transform .3s var(--n-bezier),
 background-color .3s var(--n-bezier);
 background-color: var(--n-color);
 display: flex;
 justify-content: flex-end;
`,[S(`bordered`,[k(`border`,`
 content: "";
 position: absolute;
 top: 0;
 bottom: 0;
 width: 1px;
 background-color: var(--n-border-color);
 transition: background-color .3s var(--n-bezier);
 `)]),k(`left-placement`,[S(`bordered`,[k(`border`,`
 right: 0;
 `)])]),S(`right-placement`,`
 justify-content: flex-start;
 `,[S(`bordered`,[k(`border`,`
 left: 0;
 `)]),S(`collapsed`,[O(`layout-toggle-button`,[O(`base-icon`,`
 transform: rotate(180deg);
 `)]),O(`layout-toggle-bar`,[f(`&:hover`,[k(`top`,{transform:`rotate(-12deg) scale(1.15) translateY(-2px)`}),k(`bottom`,{transform:`rotate(12deg) scale(1.15) translateY(2px)`})])])]),O(`layout-toggle-button`,`
 left: 0;
 transform: translateX(-50%) translateY(-50%);
 `,[O(`base-icon`,`
 transform: rotate(0);
 `)]),O(`layout-toggle-bar`,`
 left: -28px;
 transform: rotate(180deg);
 `,[f(`&:hover`,[k(`top`,{transform:`rotate(12deg) scale(1.15) translateY(-2px)`}),k(`bottom`,{transform:`rotate(-12deg) scale(1.15) translateY(2px)`})])])]),S(`collapsed`,[O(`layout-toggle-bar`,[f(`&:hover`,[k(`top`,{transform:`rotate(-12deg) scale(1.15) translateY(-2px)`}),k(`bottom`,{transform:`rotate(12deg) scale(1.15) translateY(2px)`})])]),O(`layout-toggle-button`,[O(`base-icon`,`
 transform: rotate(0);
 `)])]),O(`layout-toggle-button`,`
 transition:
 color .3s var(--n-bezier),
 right .3s var(--n-bezier),
 left .3s var(--n-bezier),
 border-color .3s var(--n-bezier),
 background-color .3s var(--n-bezier);
 cursor: pointer;
 width: 24px;
 height: 24px;
 position: absolute;
 top: 50%;
 right: 0;
 border-radius: 50%;
 display: flex;
 align-items: center;
 justify-content: center;
 font-size: 18px;
 color: var(--n-toggle-button-icon-color);
 border: var(--n-toggle-button-border);
 background-color: var(--n-toggle-button-color);
 box-shadow: 0 2px 4px 0px rgba(0, 0, 0, .06);
 transform: translateX(50%) translateY(-50%);
 z-index: 1;
 `,[O(`base-icon`,`
 transition: transform .3s var(--n-bezier);
 transform: rotate(180deg);
 `)]),O(`layout-toggle-bar`,`
 cursor: pointer;
 height: 72px;
 width: 32px;
 position: absolute;
 top: calc(50% - 36px);
 right: -28px;
 `,[k(`top, bottom`,`
 position: absolute;
 width: 4px;
 border-radius: 2px;
 height: 38px;
 left: 14px;
 transition: 
 background-color .3s var(--n-bezier),
 transform .3s var(--n-bezier);
 `),k(`bottom`,`
 position: absolute;
 top: 34px;
 `),f(`&:hover`,[k(`top`,{transform:`rotate(12deg) scale(1.15) translateY(-2px)`}),k(`bottom`,{transform:`rotate(-12deg) scale(1.15) translateY(2px)`})]),k(`top, bottom`,{backgroundColor:`var(--n-toggle-bar-color)`}),f(`&:hover`,[k(`top, bottom`,{backgroundColor:`var(--n-toggle-bar-color-hover)`})])]),k(`border`,`
 position: absolute;
 top: 0;
 right: 0;
 bottom: 0;
 width: 1px;
 transition: background-color .3s var(--n-bezier);
 `),O(`layout-sider-scroll-container`,`
 flex-grow: 1;
 flex-shrink: 0;
 box-sizing: border-box;
 height: 100%;
 opacity: 0;
 transition: opacity .3s var(--n-bezier);
 max-width: 100%;
 `),S(`show-content`,[O(`layout-sider-scroll-container`,{opacity:1})]),S(`absolute-positioned`,`
 position: absolute;
 left: 0;
 top: 0;
 bottom: 0;
 `)]),Bt=G({props:{clsPrefix:{type:String,required:!0},onClick:Function},render(){let{clsPrefix:e}=this;return o(`div`,{onClick:this.onClick,class:`${e}-layout-toggle-bar`},o(`div`,{class:`${e}-layout-toggle-bar__top`}),o(`div`,{class:`${e}-layout-toggle-bar__bottom`}))}}),Vt=G({name:`LayoutToggleButton`,props:{clsPrefix:{type:String,required:!0},onClick:Function},render(){let{clsPrefix:e}=this;return o(`div`,{class:`${e}-layout-toggle-button`,onClick:this.onClick},o(ae,{clsPrefix:e},{default:()=>o(Ae,null)}))}}),Ht={position:kt,bordered:Boolean,collapsedWidth:{type:Number,default:48},width:{type:[Number,String],default:272},contentClass:String,contentStyle:{type:[String,Object],default:``},collapseMode:{type:String,default:`transform`},collapsed:{type:Boolean,default:void 0},defaultCollapsed:Boolean,showCollapsedContent:{type:Boolean,default:!0},showTrigger:{type:[Boolean,String],default:!1},nativeScrollbar:{type:Boolean,default:!0},inverted:Boolean,scrollbarProps:Object,triggerClass:String,triggerStyle:[String,Object],collapsedTriggerClass:String,collapsedTriggerStyle:[String,Object],"onUpdate:collapsed":[Function,Array],onUpdateCollapsed:[Function,Array],onAfterEnter:Function,onAfterLeave:Function,onExpand:[Function,Array],onCollapse:[Function,Array],onScroll:Function},Ut=G({name:`LayoutSider`,props:Object.assign(Object.assign({},R.props),Ht),setup(e){let t=_(Mt),n=H(null),r=H(null),i=H(e.defaultCollapsed),a=Y(U(e,`collapsed`),i),o=L(()=>X(a.value?e.collapsedWidth:e.width)),s=L(()=>e.collapseMode===`transform`?{minWidth:X(e.width)}:{}),c=L(()=>t?t.siderPlacement:`left`);function l(t,i){if(e.nativeScrollbar){let{value:e}=n;e&&(i===void 0?e.scrollTo(t):e.scrollTo(t,i))}else{let{value:e}=r;e&&e.scrollTo(t,i)}}function u(){let{"onUpdate:collapsed":t,onUpdateCollapsed:n,onExpand:r,onCollapse:o}=e,{value:s}=a;n&&K(n,!s),t&&K(t,!s),i.value=!s,s?r&&K(r):o&&K(o)}let d=0,f=0,p=t=>{var n;let r=t.target;d=r.scrollLeft,f=r.scrollTop,(n=e.onScroll)==null||n.call(e,t)};ie(()=>{if(e.nativeScrollbar){let e=n.value;e&&(e.scrollTop=f,e.scrollLeft=d)}}),J(Ot,{collapsedRef:a,collapseModeRef:U(e,`collapseMode`)});let{mergedClsPrefixRef:m,inlineThemeDisabled:h}=B(e),g=R(`Layout`,`-layout-sider`,zt,Dt,e,m);function v(t){var n,r;t.propertyName===`max-width`&&(a.value?(n=e.onAfterLeave)==null||n.call(e):(r=e.onAfterEnter)==null||r.call(e))}let y={scrollTo:l},b=L(()=>{let{common:{cubicBezierEaseInOut:t},self:n}=g.value,{siderToggleButtonColor:r,siderToggleButtonBorder:i,siderToggleBarColor:a,siderToggleBarColorHover:o}=n,s={"--n-bezier":t,"--n-toggle-button-color":r,"--n-toggle-button-border":i,"--n-toggle-bar-color":a,"--n-toggle-bar-color-hover":o};return e.inverted?(s[`--n-color`]=n.siderColorInverted,s[`--n-text-color`]=n.textColorInverted,s[`--n-border-color`]=n.siderBorderColorInverted,s[`--n-toggle-button-icon-color`]=n.siderToggleButtonIconColorInverted,s.__invertScrollbar=n.__invertScrollbar):(s[`--n-color`]=n.siderColor,s[`--n-text-color`]=n.textColor,s[`--n-border-color`]=n.siderBorderColor,s[`--n-toggle-button-icon-color`]=n.siderToggleButtonIconColor),s}),x=h?I(`layout-sider`,L(()=>e.inverted?`a`:`b`),b,e):void 0;return Object.assign({scrollableElRef:n,scrollbarInstRef:r,mergedClsPrefix:m,mergedTheme:g,styleMaxWidth:o,mergedCollapsed:a,scrollContainerStyle:s,siderPlacement:c,handleNativeElScroll:p,handleTransitionend:v,handleTriggerClick:u,inlineThemeDisabled:h,cssVars:b,themeClass:x?.themeClass,onRender:x?.onRender},y)},render(){var e;let{mergedClsPrefix:t,mergedCollapsed:n,showTrigger:r}=this;return(e=this.onRender)==null||e.call(this),o(`aside`,{class:[`${t}-layout-sider`,this.themeClass,`${t}-layout-sider--${this.position}-positioned`,`${t}-layout-sider--${this.siderPlacement}-placement`,this.bordered&&`${t}-layout-sider--bordered`,n&&`${t}-layout-sider--collapsed`,(!n||this.showCollapsedContent)&&`${t}-layout-sider--show-content`],onTransitionend:this.handleTransitionend,style:[this.inlineThemeDisabled?void 0:this.cssVars,{maxWidth:this.styleMaxWidth,width:X(this.width)}]},this.nativeScrollbar?o(`div`,{class:[`${t}-layout-sider-scroll-container`,this.contentClass],onScroll:this.handleNativeElScroll,style:[this.scrollContainerStyle,{overflow:`auto`},this.contentStyle],ref:`scrollableElRef`},this.$slots):o(A,Object.assign({},this.scrollbarProps,{onScroll:this.onScroll,ref:`scrollbarInstRef`,style:this.scrollContainerStyle,contentStyle:this.contentStyle,contentClass:this.contentClass,theme:this.mergedTheme.peers.Scrollbar,themeOverrides:this.mergedTheme.peerOverrides.Scrollbar,builtinThemeOverrides:this.inverted&&this.cssVars.__invertScrollbar===`true`?{colorHover:`rgba(255, 255, 255, .4)`,color:`rgba(255, 255, 255, .3)`}:void 0}),this.$slots),r?o(r===`bar`?Bt:Vt,{clsPrefix:t,class:n?this.collapsedTriggerClass:this.triggerClass,style:n?this.collapsedTriggerStyle:this.triggerStyle,onClick:this.handleTriggerClick}):null,this.bordered?o(`div`,{class:`${t}-layout-sider__border`}):null)}}),Q=g(`n-menu`),Wt=g(`n-submenu`),Gt=g(`n-menu-item-group`),Kt=[f(`&::before`,`background-color: var(--n-item-color-hover);`),k(`arrow`,`
 color: var(--n-arrow-color-hover);
 `),k(`icon`,`
 color: var(--n-item-icon-color-hover);
 `),O(`menu-item-content-header`,`
 color: var(--n-item-text-color-hover);
 `,[f(`a`,`
 color: var(--n-item-text-color-hover);
 `),k(`extra`,`
 color: var(--n-item-text-color-hover);
 `)])],qt=[k(`icon`,`
 color: var(--n-item-icon-color-hover-horizontal);
 `),O(`menu-item-content-header`,`
 color: var(--n-item-text-color-hover-horizontal);
 `,[f(`a`,`
 color: var(--n-item-text-color-hover-horizontal);
 `),k(`extra`,`
 color: var(--n-item-text-color-hover-horizontal);
 `)])],Jt=f([O(`menu`,`
 background-color: var(--n-color);
 color: var(--n-item-text-color);
 overflow: hidden;
 transition: background-color .3s var(--n-bezier);
 box-sizing: border-box;
 font-size: var(--n-font-size);
 padding-bottom: 6px;
 `,[S(`horizontal`,`
 max-width: 100%;
 width: 100%;
 display: flex;
 overflow: hidden;
 padding-bottom: 0;
 `,[O(`submenu`,`margin: 0;`),O(`menu-item`,`margin: 0;`),O(`menu-item-content`,`
 padding: 0 20px;
 border-bottom: 2px solid #0000;
 `,[f(`&::before`,`display: none;`),S(`selected`,`border-bottom: 2px solid var(--n-border-color-horizontal)`)]),O(`menu-item-content`,[S(`selected`,[k(`icon`,`color: var(--n-item-icon-color-active-horizontal);`),O(`menu-item-content-header`,`
 color: var(--n-item-text-color-active-horizontal);
 `,[f(`a`,`color: var(--n-item-text-color-active-horizontal);`),k(`extra`,`color: var(--n-item-text-color-active-horizontal);`)])]),S(`child-active`,`
 border-bottom: 2px solid var(--n-border-color-horizontal);
 `,[O(`menu-item-content-header`,`
 color: var(--n-item-text-color-child-active-horizontal);
 `,[f(`a`,`
 color: var(--n-item-text-color-child-active-horizontal);
 `),k(`extra`,`
 color: var(--n-item-text-color-child-active-horizontal);
 `)]),k(`icon`,`
 color: var(--n-item-icon-color-child-active-horizontal);
 `)]),e(`disabled`,[e(`selected, child-active`,[f(`&:focus-within`,qt)]),S(`selected`,[$(null,[k(`icon`,`color: var(--n-item-icon-color-active-hover-horizontal);`),O(`menu-item-content-header`,`
 color: var(--n-item-text-color-active-hover-horizontal);
 `,[f(`a`,`color: var(--n-item-text-color-active-hover-horizontal);`),k(`extra`,`color: var(--n-item-text-color-active-hover-horizontal);`)])])]),S(`child-active`,[$(null,[k(`icon`,`color: var(--n-item-icon-color-child-active-hover-horizontal);`),O(`menu-item-content-header`,`
 color: var(--n-item-text-color-child-active-hover-horizontal);
 `,[f(`a`,`color: var(--n-item-text-color-child-active-hover-horizontal);`),k(`extra`,`color: var(--n-item-text-color-child-active-hover-horizontal);`)])])]),$(`border-bottom: 2px solid var(--n-border-color-horizontal);`,qt)]),O(`menu-item-content-header`,[f(`a`,`color: var(--n-item-text-color-horizontal);`)])])]),e(`responsive`,[O(`menu-item-content-header`,`
 overflow: hidden;
 text-overflow: ellipsis;
 `)]),S(`collapsed`,[O(`menu-item-content`,[S(`selected`,[f(`&::before`,`
 background-color: var(--n-item-color-active-collapsed) !important;
 `)]),O(`menu-item-content-header`,`opacity: 0;`),k(`arrow`,`opacity: 0;`),k(`icon`,`color: var(--n-item-icon-color-collapsed);`)])]),O(`menu-item`,`
 height: var(--n-item-height);
 margin-top: 6px;
 position: relative;
 `),O(`menu-item-content`,`
 box-sizing: border-box;
 line-height: 1.75;
 height: 100%;
 display: grid;
 grid-template-areas: "icon content arrow";
 grid-template-columns: auto 1fr auto;
 align-items: center;
 cursor: pointer;
 position: relative;
 padding-right: 18px;
 transition:
 background-color .3s var(--n-bezier),
 padding-left .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 `,[f(`> *`,`z-index: 1;`),f(`&::before`,`
 z-index: auto;
 content: "";
 background-color: #0000;
 position: absolute;
 left: 8px;
 right: 8px;
 top: 0;
 bottom: 0;
 pointer-events: none;
 border-radius: var(--n-border-radius);
 transition: background-color .3s var(--n-bezier);
 `),S(`disabled`,`
 opacity: .45;
 cursor: not-allowed;
 `),S(`collapsed`,[k(`arrow`,`transform: rotate(0);`)]),S(`selected`,[f(`&::before`,`background-color: var(--n-item-color-active);`),k(`arrow`,`color: var(--n-arrow-color-active);`),k(`icon`,`color: var(--n-item-icon-color-active);`),O(`menu-item-content-header`,`
 color: var(--n-item-text-color-active);
 `,[f(`a`,`color: var(--n-item-text-color-active);`),k(`extra`,`color: var(--n-item-text-color-active);`)])]),S(`child-active`,[O(`menu-item-content-header`,`
 color: var(--n-item-text-color-child-active);
 `,[f(`a`,`
 color: var(--n-item-text-color-child-active);
 `),k(`extra`,`
 color: var(--n-item-text-color-child-active);
 `)]),k(`arrow`,`
 color: var(--n-arrow-color-child-active);
 `),k(`icon`,`
 color: var(--n-item-icon-color-child-active);
 `)]),e(`disabled`,[e(`selected, child-active`,[f(`&:focus-within`,Kt)]),S(`selected`,[$(null,[k(`arrow`,`color: var(--n-arrow-color-active-hover);`),k(`icon`,`color: var(--n-item-icon-color-active-hover);`),O(`menu-item-content-header`,`
 color: var(--n-item-text-color-active-hover);
 `,[f(`a`,`color: var(--n-item-text-color-active-hover);`),k(`extra`,`color: var(--n-item-text-color-active-hover);`)])])]),S(`child-active`,[$(null,[k(`arrow`,`color: var(--n-arrow-color-child-active-hover);`),k(`icon`,`color: var(--n-item-icon-color-child-active-hover);`),O(`menu-item-content-header`,`
 color: var(--n-item-text-color-child-active-hover);
 `,[f(`a`,`color: var(--n-item-text-color-child-active-hover);`),k(`extra`,`color: var(--n-item-text-color-child-active-hover);`)])])]),S(`selected`,[$(null,[f(`&::before`,`background-color: var(--n-item-color-active-hover);`)])]),$(null,Kt)]),k(`icon`,`
 grid-area: icon;
 color: var(--n-item-icon-color);
 transition:
 color .3s var(--n-bezier),
 font-size .3s var(--n-bezier),
 margin-right .3s var(--n-bezier);
 box-sizing: content-box;
 display: inline-flex;
 align-items: center;
 justify-content: center;
 `),k(`arrow`,`
 grid-area: arrow;
 font-size: 16px;
 color: var(--n-arrow-color);
 transform: rotate(180deg);
 opacity: 1;
 transition:
 color .3s var(--n-bezier),
 transform 0.2s var(--n-bezier),
 opacity 0.2s var(--n-bezier);
 `),O(`menu-item-content-header`,`
 grid-area: content;
 transition:
 color .3s var(--n-bezier),
 opacity .3s var(--n-bezier);
 opacity: 1;
 white-space: nowrap;
 color: var(--n-item-text-color);
 `,[f(`a`,`
 outline: none;
 text-decoration: none;
 transition: color .3s var(--n-bezier);
 color: var(--n-item-text-color);
 `,[f(`&::before`,`
 content: "";
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 `)]),k(`extra`,`
 font-size: .93em;
 color: var(--n-group-text-color);
 transition: color .3s var(--n-bezier);
 `)])]),O(`submenu`,`
 cursor: pointer;
 position: relative;
 margin-top: 6px;
 `,[O(`menu-item-content`,`
 height: var(--n-item-height);
 `),O(`submenu-children`,`
 overflow: hidden;
 padding: 0;
 `,[Fe({duration:`.2s`})])]),O(`menu-item-group`,[O(`menu-item-group-title`,`
 margin-top: 6px;
 color: var(--n-group-text-color);
 cursor: default;
 font-size: .93em;
 height: 36px;
 display: flex;
 align-items: center;
 transition:
 padding-left .3s var(--n-bezier),
 color .3s var(--n-bezier);
 `)])]),O(`menu-tooltip`,[f(`a`,`
 color: inherit;
 text-decoration: none;
 `)]),O(`menu-divider`,`
 transition: background-color .3s var(--n-bezier);
 background-color: var(--n-divider-color);
 height: 1px;
 margin: 6px 18px;
 `)]);function $(e,t){return[S(`hover`,e,t),f(`&:hover`,e,t)]}var Yt=G({name:`MenuOptionContent`,props:{collapsed:Boolean,disabled:Boolean,title:[String,Function],icon:Function,extra:[String,Function],showArrow:Boolean,childActive:Boolean,hover:Boolean,paddingLeft:Number,selected:Boolean,maxIconSize:{type:Number,required:!0},activeIconSize:{type:Number,required:!0},iconMarginRight:{type:Number,required:!0},clsPrefix:{type:String,required:!0},onClick:Function,tmNode:{type:Object,required:!0},isEllipsisPlaceholder:Boolean},setup(e){let{props:t}=_(Q);return{menuProps:t,style:L(()=>{let{paddingLeft:t}=e;return{paddingLeft:t&&`${t}px`}}),iconStyle:L(()=>{let{maxIconSize:t,activeIconSize:n,iconMarginRight:r}=e;return{width:`${t}px`,height:`${t}px`,fontSize:`${n}px`,marginRight:`${r}px`}})}},render(){let{clsPrefix:e,tmNode:t,menuProps:{renderIcon:n,renderLabel:r,renderExtra:i,expandIcon:a}}=this,s=n?n(t.rawNode):Z(this.icon);return o(`div`,{onClick:e=>{var t;(t=this.onClick)==null||t.call(this,e)},role:`none`,class:[`${e}-menu-item-content`,{[`${e}-menu-item-content--selected`]:this.selected,[`${e}-menu-item-content--collapsed`]:this.collapsed,[`${e}-menu-item-content--child-active`]:this.childActive,[`${e}-menu-item-content--disabled`]:this.disabled,[`${e}-menu-item-content--hover`]:this.hover}],style:this.style},s&&o(`div`,{class:`${e}-menu-item-content__icon`,style:this.iconStyle,role:`none`},[s]),o(`div`,{class:`${e}-menu-item-content-header`,role:`none`},this.isEllipsisPlaceholder?this.title:r?r(t.rawNode):Z(this.title),this.extra||i?o(`span`,{class:`${e}-menu-item-content-header__extra`},` `,i?i(t.rawNode):Z(this.extra)):null),this.showArrow?o(ae,{ariaHidden:!0,class:`${e}-menu-item-content__arrow`,clsPrefix:e},{default:()=>a?a(t.rawNode):o(et,null)}):null)}}),Xt=8;function Zt(e){let t=_(Q),{props:n,mergedCollapsedRef:r}=t,i=_(Wt,null),a=_(Gt,null),o=L(()=>n.mode===`horizontal`),s=L(()=>o.value?n.dropdownPlacement:`tmNodes`in e?`right-start`:`right`),c=L(()=>Math.max(n.collapsedIconSize??n.iconSize,n.iconSize));return{dropdownPlacement:s,activeIconSize:L(()=>!o.value&&e.root&&r.value?n.collapsedIconSize??n.iconSize:n.iconSize),maxIconSize:c,paddingLeft:L(()=>{if(o.value)return;let{collapsedWidth:t,indent:s,rootIndent:l}=n,{root:u,isGroup:d}=e,f=l===void 0?s:l;return u?r.value?t/2-c.value/2:f:a&&typeof a.paddingLeftRef.value==`number`?s/2+a.paddingLeftRef.value:i&&typeof i.paddingLeftRef.value==`number`?(d?s/2:s)+i.paddingLeftRef.value:0}),iconMarginRight:L(()=>{let{collapsedWidth:t,indent:i,rootIndent:a}=n,{value:s}=c,{root:l}=e;return o.value||!l||!r.value?Xt:(a===void 0?i:a)+s+Xt-(t+s)/2}),NMenu:t,NSubmenu:i,NMenuOptionGroup:a}}var Qt={internalKey:{type:[String,Number],required:!0},root:Boolean,isGroup:Boolean,level:{type:Number,required:!0},title:[String,Function],extra:[String,Function]},$t=G({name:`MenuDivider`,setup(){let{mergedClsPrefixRef:e,isHorizontalRef:t}=_(Q);return()=>t.value?null:o(`div`,{class:`${e.value}-menu-divider`})}}),en=Object.assign(Object.assign({},Qt),{tmNode:{type:Object,required:!0},disabled:Boolean,icon:Function,onClick:Function}),tn=i(en),nn=G({name:`MenuOption`,props:en,setup(e){let t=Zt(e),{NSubmenu:n,NMenu:r,NMenuOptionGroup:i}=t,{props:a,mergedClsPrefixRef:o,mergedCollapsedRef:c}=r,l=n?n.mergedDisabledRef:i?i.mergedDisabledRef:{value:!1},u=L(()=>l.value||e.disabled);function d(t){let{onClick:n}=e;n&&n(t)}function f(t){u.value||(r.doSelect(e.internalKey,e.tmNode.rawNode),d(t))}return{mergedClsPrefix:o,dropdownPlacement:t.dropdownPlacement,paddingLeft:t.paddingLeft,iconMarginRight:t.iconMarginRight,maxIconSize:t.maxIconSize,activeIconSize:t.activeIconSize,mergedTheme:r.mergedThemeRef,menuProps:a,dropdownEnabled:s(()=>e.root&&c.value&&a.mode!==`horizontal`&&!u.value),selected:s(()=>r.mergedValueRef.value===e.internalKey),mergedDisabled:u,handleClick:f}},render(){let{mergedClsPrefix:e,mergedTheme:t,tmNode:n,menuProps:{renderLabel:r,nodeProps:i}}=this,a=i?.(n.rawNode);return o(`div`,Object.assign({},a,{role:`menuitem`,class:[`${e}-menu-item`,a?.class]}),o(Me,{theme:t.peers.Tooltip,themeOverrides:t.peerOverrides.Tooltip,trigger:`hover`,placement:this.dropdownPlacement,disabled:!this.dropdownEnabled||this.title===void 0,internalExtraClass:[`menu-tooltip`]},{default:()=>r?r(n.rawNode):Z(this.title),trigger:()=>o(Yt,{tmNode:n,clsPrefix:e,paddingLeft:this.paddingLeft,iconMarginRight:this.iconMarginRight,maxIconSize:this.maxIconSize,activeIconSize:this.activeIconSize,selected:this.selected,title:this.title,extra:this.extra,disabled:this.mergedDisabled,icon:this.icon,onClick:this.handleClick})}))}}),rn=Object.assign(Object.assign({},Qt),{tmNode:{type:Object,required:!0},tmNodes:{type:Array,required:!0}}),an=i(rn),on=G({name:`MenuOptionGroup`,props:rn,setup(e){let t=Zt(e),{NSubmenu:n}=t,r=L(()=>n?.mergedDisabledRef.value?!0:e.tmNode.disabled);J(Gt,{paddingLeftRef:t.paddingLeft,mergedDisabledRef:r});let{mergedClsPrefixRef:i,props:a}=_(Q);return function(){let{value:n}=i,r=t.paddingLeft.value,{nodeProps:s}=a,c=s?.(e.tmNode.rawNode);return o(`div`,{class:`${n}-menu-item-group`,role:`group`},o(`div`,Object.assign({},c,{class:[`${n}-menu-item-group-title`,c?.class],style:[c?.style||``,r===void 0?``:`padding-left: ${r}px;`]}),Z(e.title),e.extra?o(P,null,` `,Z(e.extra)):null),o(`div`,null,e.tmNodes.map(e=>ln(e,a))))}}});function sn(e){return e.type===`divider`||e.type===`render`}function cn(e){return e.type===`divider`}function ln(e,t){let{rawNode:n}=e,{show:r}=n;if(r===!1)return null;if(sn(n))return cn(n)?o($t,Object.assign({key:e.key},n.props)):null;let{labelField:i}=t,{key:a,level:s,isGroup:c}=e,l=Object.assign(Object.assign({},n),{title:n.title||n[i],extra:n.titleExtra||n.extra,key:a,internalKey:a,level:s,root:s===0,isGroup:c});return e.children?e.isGroup?o(on,Te(l,an,{tmNode:e,tmNodes:e.children,key:a})):o(fn,Te(l,dn,{key:a,rawNodes:n[t.childrenField],tmNodes:e.children,tmNode:e})):o(nn,Te(l,tn,{key:a,tmNode:e}))}var un=Object.assign(Object.assign({},Qt),{rawNodes:{type:Array,default:()=>[]},tmNodes:{type:Array,default:()=>[]},tmNode:{type:Object,required:!0},disabled:Boolean,icon:Function,onClick:Function,domId:String,virtualChildActive:{type:Boolean,default:void 0},isEllipsisPlaceholder:Boolean}),dn=i(un),fn=G({name:`Submenu`,props:un,setup(e){let t=Zt(e),{NMenu:n,NSubmenu:r}=t,{props:i,mergedCollapsedRef:a,mergedThemeRef:o}=n,c=L(()=>{let{disabled:t}=e;return r?.mergedDisabledRef.value||i.disabled?!0:t}),l=H(!1);J(Wt,{paddingLeftRef:t.paddingLeft,mergedDisabledRef:c}),J(Gt,null);function u(){let{onClick:t}=e;t&&t()}function d(){c.value||(a.value||n.toggleExpand(e.internalKey),u())}function f(e){l.value=e}return{menuProps:i,mergedTheme:o,doSelect:n.doSelect,inverted:n.invertedRef,isHorizontal:n.isHorizontalRef,mergedClsPrefix:n.mergedClsPrefixRef,maxIconSize:t.maxIconSize,activeIconSize:t.activeIconSize,iconMarginRight:t.iconMarginRight,dropdownPlacement:t.dropdownPlacement,dropdownShow:l,paddingLeft:t.paddingLeft,mergedDisabled:c,mergedValue:n.mergedValueRef,childActive:s(()=>e.virtualChildActive??n.activePathRef.value.includes(e.internalKey)),collapsed:L(()=>i.mode===`horizontal`?!1:a.value?!0:!n.mergedExpandedKeysRef.value.includes(e.internalKey)),dropdownEnabled:L(()=>!c.value&&(i.mode===`horizontal`||a.value)),handlePopoverShowChange:f,handleClick:d}},render(){let{mergedClsPrefix:e,menuProps:{renderIcon:t,renderLabel:n}}=this,r=()=>{let{isHorizontal:e,paddingLeft:t,collapsed:n,mergedDisabled:r,maxIconSize:i,activeIconSize:a,title:s,childActive:c,icon:l,handleClick:u,menuProps:{nodeProps:d},dropdownShow:f,iconMarginRight:p,tmNode:m,mergedClsPrefix:h,isEllipsisPlaceholder:g,extra:_}=this,v=d?.(m.rawNode);return o(`div`,Object.assign({},v,{class:[`${h}-menu-item`,v?.class],role:`menuitem`}),o(Yt,{tmNode:m,paddingLeft:t,collapsed:n,disabled:r,iconMarginRight:p,maxIconSize:i,activeIconSize:a,title:s,extra:_,showArrow:!e,childActive:c,clsPrefix:h,icon:l,hover:f,onClick:u,isEllipsisPlaceholder:g}))},i=()=>o(E,null,{default:()=>{let{tmNodes:t,collapsed:n}=this;return n?null:o(`div`,{class:`${e}-submenu-children`,role:`menu`},t.map(e=>ln(e,this.menuProps)))}});return this.root?o(je,Object.assign({size:`large`,trigger:`hover`},this.menuProps?.dropdownProps,{themeOverrides:this.mergedTheme.peerOverrides.Dropdown,theme:this.mergedTheme.peers.Dropdown,builtinThemeOverrides:{fontSizeLarge:`14px`,optionIconSizeLarge:`18px`},value:this.mergedValue,disabled:!this.dropdownEnabled,placement:this.dropdownPlacement,keyField:this.menuProps.keyField,labelField:this.menuProps.labelField,childrenField:this.menuProps.childrenField,onUpdateShow:this.handlePopoverShowChange,options:this.rawNodes,onSelect:this.doSelect,inverted:this.inverted,renderIcon:t,renderLabel:n}),{default:()=>o(`div`,{class:`${e}-submenu`,role:`menu`,"aria-expanded":!this.collapsed,id:this.domId},r(),this.isHorizontal?null:i())}):o(`div`,{class:`${e}-submenu`,role:`menu`,"aria-expanded":!this.collapsed,id:this.domId},r(),i())}}),pn=G({name:`Menu`,inheritAttrs:!1,props:Object.assign(Object.assign({},R.props),{options:{type:Array,default:()=>[]},collapsed:{type:Boolean,default:void 0},collapsedWidth:{type:Number,default:48},iconSize:{type:Number,default:20},collapsedIconSize:{type:Number,default:24},rootIndent:Number,indent:{type:Number,default:32},labelField:{type:String,default:`label`},keyField:{type:String,default:`key`},childrenField:{type:String,default:`children`},disabledField:{type:String,default:`disabled`},defaultExpandAll:Boolean,defaultExpandedKeys:Array,expandedKeys:Array,value:[String,Number],defaultValue:{type:[String,Number],default:null},mode:{type:String,default:`vertical`},watchProps:{type:Array,default:void 0},disabled:Boolean,show:{type:Boolean,default:!0},inverted:Boolean,"onUpdate:expandedKeys":[Function,Array],onUpdateExpandedKeys:[Function,Array],onUpdateValue:[Function,Array],"onUpdate:value":[Function,Array],expandIcon:Function,renderIcon:Function,renderLabel:Function,renderExtra:Function,dropdownProps:Object,accordion:Boolean,nodeProps:Function,dropdownPlacement:{type:String,default:`bottom`},responsive:Boolean,items:Array,onOpenNamesChange:[Function,Array],onSelect:[Function,Array],onExpandedNamesChange:[Function,Array],expandedNames:Array,defaultExpandedNames:Array}),setup(e){let{mergedClsPrefixRef:t,inlineThemeDisabled:n}=B(e),r=R(`Menu`,`-menu`,Jt,We,e,t),i=_(Ot,null),a=L(()=>{let{collapsed:t}=e;if(t!==void 0)return t;if(i){let{collapseModeRef:e,collapsedRef:t}=i;if(e.value===`width`)return t.value??!1}return!1}),s=L(()=>{let{keyField:t,childrenField:n,disabledField:r}=e;return ge(e.items||e.options,{getIgnored(e){return sn(e)},getChildren(e){return e[n]},getDisabled(e){return e[r]},getKey(e){return e[t]??e.name}})}),c=L(()=>new Set(s.value.treeNodes.map(e=>e.key))),{watchProps:l}=e,u=H(null);l?.includes(`defaultValue`)?p(()=>{u.value=e.defaultValue}):u.value=e.defaultValue;let d=Y(U(e,`value`),u),f=H([]),m=()=>{f.value=e.defaultExpandAll?s.value.getNonLeafKeys():e.defaultExpandedNames||e.defaultExpandedKeys||s.value.getPath(d.value,{includeSelf:!1}).keyPath};l?.includes(`defaultExpandedKeys`)?p(m):m();let h=De(e,[`expandedNames`,`expandedKeys`]),g=Y(h,f),v=L(()=>s.value.treeNodes),y=L(()=>s.value.getPath(d.value).keyPath);J(Q,{props:e,mergedCollapsedRef:a,mergedThemeRef:r,mergedValueRef:d,mergedExpandedKeysRef:g,activePathRef:y,mergedClsPrefixRef:t,isHorizontalRef:L(()=>e.mode===`horizontal`),invertedRef:U(e,`inverted`),doSelect:b,toggleExpand:S});function b(t,n){let{"onUpdate:value":r,onUpdateValue:i,onSelect:a}=e;i&&K(i,t,n),r&&K(r,t,n),a&&K(a,t,n),u.value=t}function x(t){let{"onUpdate:expandedKeys":n,onUpdateExpandedKeys:r,onExpandedNamesChange:i,onOpenNamesChange:a}=e;n&&K(n,t),r&&K(r,t),i&&K(i,t),a&&K(a,t),f.value=t}function S(t){let n=Array.from(g.value),r=n.findIndex(e=>e===t);if(~r)n.splice(r,1);else{if(e.accordion&&c.value.has(t)){let e=n.findIndex(e=>c.value.has(e));e>-1&&n.splice(e,1)}n.push(t)}x(n)}let C=t=>{let n=s.value.getPath(t??d.value,{includeSelf:!1}).keyPath;if(!n.length)return;let r=Array.from(g.value),i=new Set([...r,...n]);e.accordion&&c.value.forEach(e=>{i.has(e)&&!n.includes(e)&&i.delete(e)}),x(Array.from(i))},w=L(()=>{let{inverted:t}=e,{common:{cubicBezierEaseInOut:n},self:i}=r.value,{borderRadius:a,borderColorHorizontal:o,fontSize:s,itemHeight:c,dividerColor:l}=i,u={"--n-divider-color":l,"--n-bezier":n,"--n-font-size":s,"--n-border-color-horizontal":o,"--n-border-radius":a,"--n-item-height":c};return t?(u[`--n-group-text-color`]=i.groupTextColorInverted,u[`--n-color`]=i.colorInverted,u[`--n-item-text-color`]=i.itemTextColorInverted,u[`--n-item-text-color-hover`]=i.itemTextColorHoverInverted,u[`--n-item-text-color-active`]=i.itemTextColorActiveInverted,u[`--n-item-text-color-child-active`]=i.itemTextColorChildActiveInverted,u[`--n-item-text-color-child-active-hover`]=i.itemTextColorChildActiveInverted,u[`--n-item-text-color-active-hover`]=i.itemTextColorActiveHoverInverted,u[`--n-item-icon-color`]=i.itemIconColorInverted,u[`--n-item-icon-color-hover`]=i.itemIconColorHoverInverted,u[`--n-item-icon-color-active`]=i.itemIconColorActiveInverted,u[`--n-item-icon-color-active-hover`]=i.itemIconColorActiveHoverInverted,u[`--n-item-icon-color-child-active`]=i.itemIconColorChildActiveInverted,u[`--n-item-icon-color-child-active-hover`]=i.itemIconColorChildActiveHoverInverted,u[`--n-item-icon-color-collapsed`]=i.itemIconColorCollapsedInverted,u[`--n-item-text-color-horizontal`]=i.itemTextColorHorizontalInverted,u[`--n-item-text-color-hover-horizontal`]=i.itemTextColorHoverHorizontalInverted,u[`--n-item-text-color-active-horizontal`]=i.itemTextColorActiveHorizontalInverted,u[`--n-item-text-color-child-active-horizontal`]=i.itemTextColorChildActiveHorizontalInverted,u[`--n-item-text-color-child-active-hover-horizontal`]=i.itemTextColorChildActiveHoverHorizontalInverted,u[`--n-item-text-color-active-hover-horizontal`]=i.itemTextColorActiveHoverHorizontalInverted,u[`--n-item-icon-color-horizontal`]=i.itemIconColorHorizontalInverted,u[`--n-item-icon-color-hover-horizontal`]=i.itemIconColorHoverHorizontalInverted,u[`--n-item-icon-color-active-horizontal`]=i.itemIconColorActiveHorizontalInverted,u[`--n-item-icon-color-active-hover-horizontal`]=i.itemIconColorActiveHoverHorizontalInverted,u[`--n-item-icon-color-child-active-horizontal`]=i.itemIconColorChildActiveHorizontalInverted,u[`--n-item-icon-color-child-active-hover-horizontal`]=i.itemIconColorChildActiveHoverHorizontalInverted,u[`--n-arrow-color`]=i.arrowColorInverted,u[`--n-arrow-color-hover`]=i.arrowColorHoverInverted,u[`--n-arrow-color-active`]=i.arrowColorActiveInverted,u[`--n-arrow-color-active-hover`]=i.arrowColorActiveHoverInverted,u[`--n-arrow-color-child-active`]=i.arrowColorChildActiveInverted,u[`--n-arrow-color-child-active-hover`]=i.arrowColorChildActiveHoverInverted,u[`--n-item-color-hover`]=i.itemColorHoverInverted,u[`--n-item-color-active`]=i.itemColorActiveInverted,u[`--n-item-color-active-hover`]=i.itemColorActiveHoverInverted,u[`--n-item-color-active-collapsed`]=i.itemColorActiveCollapsedInverted):(u[`--n-group-text-color`]=i.groupTextColor,u[`--n-color`]=i.color,u[`--n-item-text-color`]=i.itemTextColor,u[`--n-item-text-color-hover`]=i.itemTextColorHover,u[`--n-item-text-color-active`]=i.itemTextColorActive,u[`--n-item-text-color-child-active`]=i.itemTextColorChildActive,u[`--n-item-text-color-child-active-hover`]=i.itemTextColorChildActiveHover,u[`--n-item-text-color-active-hover`]=i.itemTextColorActiveHover,u[`--n-item-icon-color`]=i.itemIconColor,u[`--n-item-icon-color-hover`]=i.itemIconColorHover,u[`--n-item-icon-color-active`]=i.itemIconColorActive,u[`--n-item-icon-color-active-hover`]=i.itemIconColorActiveHover,u[`--n-item-icon-color-child-active`]=i.itemIconColorChildActive,u[`--n-item-icon-color-child-active-hover`]=i.itemIconColorChildActiveHover,u[`--n-item-icon-color-collapsed`]=i.itemIconColorCollapsed,u[`--n-item-text-color-horizontal`]=i.itemTextColorHorizontal,u[`--n-item-text-color-hover-horizontal`]=i.itemTextColorHoverHorizontal,u[`--n-item-text-color-active-horizontal`]=i.itemTextColorActiveHorizontal,u[`--n-item-text-color-child-active-horizontal`]=i.itemTextColorChildActiveHorizontal,u[`--n-item-text-color-child-active-hover-horizontal`]=i.itemTextColorChildActiveHoverHorizontal,u[`--n-item-text-color-active-hover-horizontal`]=i.itemTextColorActiveHoverHorizontal,u[`--n-item-icon-color-horizontal`]=i.itemIconColorHorizontal,u[`--n-item-icon-color-hover-horizontal`]=i.itemIconColorHoverHorizontal,u[`--n-item-icon-color-active-horizontal`]=i.itemIconColorActiveHorizontal,u[`--n-item-icon-color-active-hover-horizontal`]=i.itemIconColorActiveHoverHorizontal,u[`--n-item-icon-color-child-active-horizontal`]=i.itemIconColorChildActiveHorizontal,u[`--n-item-icon-color-child-active-hover-horizontal`]=i.itemIconColorChildActiveHoverHorizontal,u[`--n-arrow-color`]=i.arrowColor,u[`--n-arrow-color-hover`]=i.arrowColorHover,u[`--n-arrow-color-active`]=i.arrowColorActive,u[`--n-arrow-color-active-hover`]=i.arrowColorActiveHover,u[`--n-arrow-color-child-active`]=i.arrowColorChildActive,u[`--n-arrow-color-child-active-hover`]=i.arrowColorChildActiveHover,u[`--n-item-color-hover`]=i.itemColorHover,u[`--n-item-color-active`]=i.itemColorActive,u[`--n-item-color-active-hover`]=i.itemColorActiveHover,u[`--n-item-color-active-collapsed`]=i.itemColorActiveCollapsed),u}),T=n?I(`menu`,L(()=>e.inverted?`a`:`b`),w,e):void 0,E=_e(),D=H(null),O=H(null),k=!0,A=()=>{var e;k?k=!1:(e=D.value)==null||e.sync({showAllItemsBeforeCalculate:!0})};function j(){return document.getElementById(E)}let M=H(-1);function N(t){M.value=e.options.length-t}function ee(e){e||(M.value=-1)}let te=L(()=>{let t=M.value;return{children:t===-1?[]:e.options.slice(t)}}),ne=L(()=>{let{childrenField:t,disabledField:n,keyField:r}=e;return ge([te.value],{getIgnored(e){return sn(e)},getChildren(e){return e[t]},getDisabled(e){return e[n]},getKey(e){return e[r]??e.name}})}),P=L(()=>ge([{}]).treeNodes[0]);function F(){if(M.value===-1)return o(fn,{root:!0,level:0,key:`__ellpisisGroupPlaceholder__`,internalKey:`__ellpisisGroupPlaceholder__`,title:`···`,tmNode:P.value,domId:E,isEllipsisPlaceholder:!0});let e=ne.value.treeNodes[0],t=y.value;return o(fn,{level:0,root:!0,key:`__ellpisisGroup__`,internalKey:`__ellpisisGroup__`,title:`···`,virtualChildActive:!!e.children?.some(e=>t.includes(e.key)),tmNode:e,domId:E,rawNodes:e.rawNode.children||[],tmNodes:e.children||[],isEllipsisPlaceholder:!0})}return{mergedClsPrefix:t,controlledExpandedKeys:h,uncontrolledExpanededKeys:f,mergedExpandedKeys:g,uncontrolledValue:u,mergedValue:d,activePath:y,tmNodes:v,mergedTheme:r,mergedCollapsed:a,cssVars:n?void 0:w,themeClass:T?.themeClass,overflowRef:D,counterRef:O,updateCounter:()=>{},onResize:A,onUpdateOverflow:ee,onUpdateCount:N,renderCounter:F,getCounter:j,onRender:T?.onRender,showOption:C,deriveResponsiveState:A}},render(){let{mergedClsPrefix:e,mode:t,themeClass:r,onRender:i}=this;i?.();let a=()=>this.tmNodes.map(e=>ln(e,this.$props)),s=t===`horizontal`&&this.responsive,c=()=>o(`div`,n(this.$attrs,{role:t===`horizontal`?`menubar`:`menu`,class:[`${e}-menu`,r,`${e}-menu--${t}`,s&&`${e}-menu--responsive`,this.mergedCollapsed&&`${e}-menu--collapsed`],style:this.cssVars}),s?o(he,{ref:`overflowRef`,onUpdateOverflow:this.onUpdateOverflow,getCounter:this.getCounter,onUpdateCount:this.onUpdateCount,updateCounter:this.updateCounter,style:{width:`100%`,display:`flex`,overflow:`hidden`}},{default:a,counter:this.renderCounter}):a());return s?o(v,{onResize:this.onResize},{default:c}):c()}}),mn={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},hn=G({name:`ArchiveOutline`,render:function(e,t){return C(),N(`svg`,mn,t[0]||=[j(`path`,{d:`M80 152v256a40.12 40.12 0 0 0 40 40h272a40.12 40.12 0 0 0 40-40V152`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),j(`rect`,{x:`48`,y:`64`,width:`416`,height:`80`,rx:`28`,ry:`28`,fill:`none`,stroke:`currentColor`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),j(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M320 304l-64 64l-64-64`},null,-1),j(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M256 345.89V224`},null,-1)])}}),gn={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},_n=G({name:`CubeOutline`,render:function(e,t){return C(),N(`svg`,gn,t[0]||=[j(`path`,{d:`M448 341.37V170.61A32 32 0 0 0 432.11 143l-152-88.46a47.94 47.94 0 0 0-48.24 0L79.89 143A32 32 0 0 0 64 170.61v170.76A32 32 0 0 0 79.89 369l152 88.46a48 48 0 0 0 48.24 0l152-88.46A32 32 0 0 0 448 341.37z`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),j(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M69 153.99l187 110l187-110`},null,-1),j(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M256 463.99v-200`},null,-1)])}}),vn={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},yn=G({name:`DocumentTextOutline`,render:function(e,t){return C(),N(`svg`,vn,t[0]||=[j(`path`,{d:`M416 221.25V416a48 48 0 0 1-48 48H144a48 48 0 0 1-48-48V96a48 48 0 0 1 48-48h98.75a32 32 0 0 1 22.62 9.37l141.26 141.26a32 32 0 0 1 9.37 22.62z`,fill:`none`,stroke:`currentColor`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),j(`path`,{d:`M256 56v120a32 32 0 0 0 32 32h120`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),j(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M176 288h160`},null,-1),j(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M176 368h160`},null,-1)])}}),bn={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},xn=G({name:`EarthOutline`,render:function(e,t){return C(),N(`svg`,bn,t[0]||=[ue(`<path d="M464 256c0-114.87-93.13-208-208-208S48 141.13 48 256s93.13 208 208 208s208-93.13 208-208z" fill="none" stroke="currentColor" stroke-miterlimit="10" stroke-width="32"></path><path d="M445.57 172.14c-16.06.1-14.48 29.73-34.49 15.75c-7.43-5.18-12-12.71-21.33-15c-8.15-2-16.5.08-24.55 1.47c-9.15 1.58-20 2.29-26.94 9.22c-6.71 6.67-10.26 15.62-17.4 22.33c-13.81 13-19.64 27.19-10.7 45.57c8.6 17.67 26.59 27.26 46 26c19.07-1.27 38.88-12.33 38.33 15.38c-.2 9.8 1.85 16.6 4.86 25.71c2.79 8.4 2.6 16.54 3.24 25.21c1.18 16.2 4.16 34.36 12.2 48.67l15-21.16c1.85-2.62 5.72-6.29 6.64-9.38c1.63-5.47-1.58-14.87-1.95-21s-.19-12.34-1.13-18.47c-1.32-8.59-6.4-16.64-7.1-25.13c-1.29-15.81 1.6-28.43-10.58-41.65c-11.76-12.75-29-15.81-45.47-13.22c-8.3 1.3-41.71 6.64-28.3-12.33c2.65-3.73 7.28-6.79 10.26-10.34c2.59-3.09 4.84-8.77 7.88-11.18s17-5.18 21-3.95s8.17 7 11.64 9.56a49.89 49.89 0 0 0 21.81 9.36c13.66 2 42.22-5.94 42-23.46c-.04-8.4-7.84-20.1-10.92-27.96z" fill="currentColor"></path><path d="M287.45 316.3c-5.33-22.44-35.82-29.94-52.26-42.11c-9.45-7-17.86-17.81-30.27-18.69c-5.72-.41-10.51.83-16.18-.64c-5.2-1.34-9.28-4.14-14.82-3.41c-10.35 1.36-16.88 12.42-28 10.92c-10.55-1.42-21.42-13.76-23.82-23.81c-3.08-12.92 7.14-17.11 18.09-18.26c4.57-.48 9.7-1 14.09.67c5.78 2.15 8.51 7.81 13.7 10.67c9.73 5.33 11.7-3.19 10.21-11.83c-2.23-12.94-4.83-18.22 6.71-27.12c8-6.14 14.84-10.58 13.56-21.61c-.76-6.48-4.31-9.41-1-15.86c2.51-4.91 9.4-9.34 13.89-12.27c11.59-7.56 49.65-7 34.1-28.16c-4.57-6.21-13-17.31-21-18.83c-10-1.89-14.44 9.27-21.41 14.19c-7.2 5.09-21.22 10.87-28.43 3c-9.7-10.59 6.43-14.07 10-21.46s-8.27-21.36-14.61-24.9l-29.81 33.43a41.52 41.52 0 0 0 8.34 31.86c5.93 7.63 15.37 10.08 15.8 20.5c.42 10-1.14 15.12-7.68 22.15c-2.83 3-4.83 7.26-7.71 10.07c-3.53 3.43-2.22 2.38-7.73 3.32c-10.36 1.75-19.18 4.45-29.19 7.21C95.34 199.94 93.8 172.69 86.2 162l-25 20.19c-.27 3.31 4.1 9.4 5.29 13c6.83 20.57 20.61 36.48 29.51 56.16c9.37 20.84 34.53 15.06 45.64 33.32c9.86 16.2-.67 36.71 6.71 53.67c5.36 12.31 18 15 26.72 24c8.91 9.09 8.72 21.53 10.08 33.36a305.22 305.22 0 0 0 7.45 41.28c1.21 4.69 2.32 10.89 5.53 14.76c2.2 2.66 9.75 4.95 6.7 5.83c4.26.7 11.85 4.68 15.4 1.76c4.68-3.84 3.43-15.66 4.24-21c2.43-15.9 10.39-31.45 21.13-43.35c10.61-11.74 25.15-19.69 34.11-33c8.73-12.98 11.36-30.49 7.74-45.68zm-33.39 26.32c-6 10.71-19.36 17.88-27.95 26.39c-2.33 2.31-7.29 10.31-10.21 8.58c-2.09-1.24-2.8-11.62-3.57-14a61.17 61.17 0 0 0-21.71-29.95c-3.13-2.37-10.89-5.45-12.68-8.7c-2-3.53-.2-11.86-.13-15.7c.11-5.6-2.44-14.91-1.06-20c1.6-5.87-1.48-2.33 3.77-3.49c2.77-.62 14.21 1.39 17.66 2.11c5.48 1.14 8.5 4.55 12.82 8c11.36 9.11 23.87 16.16 36.6 23.14c9.86 5.46 12.76 12.37 6.46 23.62z" fill="currentColor"></path><path d="M184.46 67.09c4.74 4.63 9.2 10.11 16.27 10.57c6.69.45 13-3.17 18.84 1.38c6.48 5 11.15 11.33 19.75 12.89c8.32 1.51 17.13-3.35 19.19-11.86c2-8.11-2.31-16.93-2.57-25.07c0-1.13.61-6.15-.17-7c-.58-.64-5.42.08-6.16.1q-8.13.24-16.22 1.12a207.1 207.1 0 0 0-57.18 14.65c2.43 1.68 5.48 2.35 8.25 3.22z" fill="currentColor"></path><path d="M356.4 123.27c8.49 0 17.11-3.8 14.37-13.62c-2.3-8.23-6.22-17.16-15.76-12.72c-6.07 2.82-14.67 10-15.38 17.12c-.81 8.08 11.11 9.22 16.77 9.22z" fill="currentColor"></path><path d="M349.62 166.24c8.67 5.19 21.53 2.75 28.07-4.66c5.11-5.8 8.12-15.87 17.31-15.86a15.4 15.4 0 0 1 10.82 4.41c3.8 3.93 3.05 7.62 3.86 12.54c1.81 11.05 13.66.63 16.75-3.65c2-2.79 4.71-6.93 3.8-10.56c-.84-3.39-4.8-7-6.56-10.11c-5.14-9-9.37-19.47-17.07-26.74c-7.41-7-16.52-6.19-23.55 1.08c-5.76 6-12.45 10.75-16.39 18.05c-2.78 5.13-5.91 7.58-11.54 8.91c-3.1.73-6.64 1-9.24 3.08c-7.24 5.7-3.12 19.39 3.74 23.51z" fill="currentColor"></path>`,6)])}}),Sn={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},Cn=G({name:`GitNetworkOutline`,render:function(e,t){return C(),N(`svg`,Sn,t[0]||=[ue(`<circle cx="128" cy="96" r="48" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"></circle><circle cx="256" cy="416" r="48" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"></circle><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M256 256v112"></path><circle cx="384" cy="96" r="48" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"></circle><path d="M128 144c0 74.67 68.92 112 128 112" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"></path><path d="M384 144c0 74.67-68.92 112-128 112" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"></path>`,6)])}}),wn={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},Tn=G({name:`GridOutline`,render:function(e,t){return C(),N(`svg`,wn,t[0]||=[j(`rect`,{x:`48`,y:`48`,width:`176`,height:`176`,rx:`20`,ry:`20`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),j(`rect`,{x:`288`,y:`48`,width:`176`,height:`176`,rx:`20`,ry:`20`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),j(`rect`,{x:`48`,y:`288`,width:`176`,height:`176`,rx:`20`,ry:`20`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),j(`rect`,{x:`288`,y:`288`,width:`176`,height:`176`,rx:`20`,ry:`20`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1)])}}),En={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},Dn=G({name:`LogOutOutline`,render:function(e,t){return C(),N(`svg`,En,t[0]||=[j(`path`,{d:`M304 336v40a40 40 0 0 1-40 40H104a40 40 0 0 1-40-40V136a40 40 0 0 1 40-40h152c22.09 0 48 17.91 48 40v40`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),j(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M368 336l80-80l-80-80`},null,-1),j(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M176 256h256`},null,-1)])}}),On={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},kn=G({name:`MenuOutline`,render:function(e,t){return C(),N(`svg`,On,t[0]||=[j(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-miterlimit":`10`,"stroke-width":`32`,d:`M80 160h352`},null,-1),j(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-miterlimit":`10`,"stroke-width":`32`,d:`M80 256h352`},null,-1),j(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-miterlimit":`10`,"stroke-width":`32`,d:`M80 352h352`},null,-1)])}}),An={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},jn=G({name:`PulseOutline`,render:function(e,t){return C(),N(`svg`,An,t[0]||=[j(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M48 320h64l64-256l64 384l64-224l32 96h64`},null,-1),j(`circle`,{cx:`432`,cy:`320`,r:`32`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1)])}}),Mn={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},Nn=G({name:`SettingsOutline`,render:function(e,t){return C(),N(`svg`,Mn,t[0]||=[j(`path`,{d:`M262.29 192.31a64 64 0 1 0 57.4 57.4a64.13 64.13 0 0 0-57.4-57.4zM416.39 256a154.34 154.34 0 0 1-1.53 20.79l45.21 35.46a10.81 10.81 0 0 1 2.45 13.75l-42.77 74a10.81 10.81 0 0 1-13.14 4.59l-44.9-18.08a16.11 16.11 0 0 0-15.17 1.75A164.48 164.48 0 0 1 325 400.8a15.94 15.94 0 0 0-8.82 12.14l-6.73 47.89a11.08 11.08 0 0 1-10.68 9.17h-85.54a11.11 11.11 0 0 1-10.69-8.87l-6.72-47.82a16.07 16.07 0 0 0-9-12.22a155.3 155.3 0 0 1-21.46-12.57a16 16 0 0 0-15.11-1.71l-44.89 18.07a10.81 10.81 0 0 1-13.14-4.58l-42.77-74a10.8 10.8 0 0 1 2.45-13.75l38.21-30a16.05 16.05 0 0 0 6-14.08c-.36-4.17-.58-8.33-.58-12.5s.21-8.27.58-12.35a16 16 0 0 0-6.07-13.94l-38.19-30A10.81 10.81 0 0 1 49.48 186l42.77-74a10.81 10.81 0 0 1 13.14-4.59l44.9 18.08a16.11 16.11 0 0 0 15.17-1.75A164.48 164.48 0 0 1 187 111.2a15.94 15.94 0 0 0 8.82-12.14l6.73-47.89A11.08 11.08 0 0 1 213.23 42h85.54a11.11 11.11 0 0 1 10.69 8.87l6.72 47.82a16.07 16.07 0 0 0 9 12.22a155.3 155.3 0 0 1 21.46 12.57a16 16 0 0 0 15.11 1.71l44.89-18.07a10.81 10.81 0 0 1 13.14 4.58l42.77 74a10.8 10.8 0 0 1-2.45 13.75l-38.21 30a16.05 16.05 0 0 0-6.05 14.08c.33 4.14.55 8.3.55 12.47z`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1)])}}),Pn={key:0,class:`brand-copy`},Fn={class:`mobile-drawer-account`},In={class:`account-copy`},Ln={class:`app-header-leading`},Rn={class:`app-title`},zn={class:`account`},Bn={class:`account-copy`},Vn={class:`update-banner-body`},Hn=[`href`],Un=G({__name:`AppLayout`,setup(e){let t=Ke(),n=Ue(),i=qe(),a=$e(),s=Oe(),d=Ne(),f=H(!1),p=H(window.innerWidth),m=L(()=>Math.min(320,Math.round(p.value*.86))),h=H(window.innerWidth<1100);function g(e,t,n){return{label:()=>o(Ge,{to:{name:t}},{default:()=>e}),key:t,icon:()=>o(Le,null,{default:()=>o(n)})}}let _=[g(`运行概览`,`dashboard`,Tn),g(`代理编排`,`orchestration`,Cn),g(`配置管理`,`config`,yn),g(`配置能力`,`schema`,Ye),g(`dae 版本`,`versions`,_n),g(`Geo 数据`,`geo`,xn),g(`故障诊断`,`diagnostics`,jn),g(`连接活动`,`connections`,Ze),g(`运行日志`,`logs`,Xe),g(`配置备份`,`backups`,hn),g(`面板设置`,`settings`,Nn)],v=L(()=>String(t.name||`dashboard`)),y=L(()=>String(t.meta.title||`kdae-panel`));async function x(){try{await i.logout(),await n.replace({name:`login`})}catch(e){s.error(e instanceof Error?e.message:`退出登录失败`)}}function S(){i.clearSession(),n.replace({name:`login`}),s.warning(`登录会话已过期，请重新登录`)}function w(){p.value=window.innerWidth,!d.value&&window.innerWidth<1100&&(h.value=!0)}c(d,()=>{f.value=!1});let E=H(null),D=H(!1);async function O(){try{E.value=await re(`/api/v1/panel/update`)}catch{E.value=null}}function k(e){let t=e.detail;E.value&&t&&(E.value.status=t)}return u(()=>{window.addEventListener(`kdae-panel:auth-expired`,S),window.addEventListener(`kdae-panel:self-update-changed`,k),window.addEventListener(`resize`,w),a.ensure(),O()}),b(()=>{window.removeEventListener(`kdae-panel:auth-expired`,S),window.removeEventListener(`kdae-panel:self-update-changed`,k),window.removeEventListener(`resize`,w)}),(e,t)=>{let n=l(`RouterView`);return C(),W(z(Pt),{"has-sider":!z(d),class:`app-shell`},{default:V(()=>[z(d)?q(``,!0):(C(),W(z(Ut),{key:0,bordered:``,"collapse-mode":`width`,"collapsed-width":64,width:236,collapsed:h.value,"show-trigger":`bar`,onCollapse:t[0]||=e=>h.value=!0,onExpand:t[1]||=e=>h.value=!1},{default:V(()=>[j(`div`,{class:ce([`brand`,{compact:h.value}])},[t[7]||=j(`div`,{class:`brand-mark`},`K`,-1),h.value?q(``,!0):(C(),N(`div`,Pn,[...t[6]||=[j(`strong`,null,`kdae-panel`,-1),j(`span`,null,`零侵入管理面板`,-1)]]))],2),r(z(pn),{value:v.value,collapsed:h.value,"collapsed-width":64,"collapsed-icon-size":22,options:_},null,8,[`value`,`collapsed`])]),_:1},8,[`collapsed`])),r(z(wt),{show:f.value,"onUpdate:show":t[3]||=e=>f.value=e,placement:`left`,width:m.value},{default:V(()=>[r(z(Tt),{class:`mobile-nav-drawer`,"native-scrollbar":!1,"body-content-style":`padding: 0;`},{footer:V(()=>[j(`div`,Fn,[r(z(lt),{round:``,size:`small`},{default:V(()=>[T(F(z(i).user?.username?.slice(0,1).toUpperCase()),1)]),_:1}),j(`div`,In,[j(`strong`,null,F(z(i).user?.username),1),t[8]||=j(`span`,null,`管理员`,-1)]),r(z(ne),{quaternary:``,circle:``,title:`退出登录`,"aria-label":`退出登录`,onClick:x},{icon:V(()=>[r(z(Le),null,{default:V(()=>[r(z(Dn))]),_:1})]),_:1})])]),default:V(()=>[t[9]||=j(`div`,{class:`brand mobile-drawer-brand`},[j(`div`,{class:`brand-mark`},`K`),j(`div`,{class:`brand-copy`},[j(`strong`,null,`kdae-panel`),j(`span`,null,`零侵入管理面板`)])],-1),r(z(pn),{value:v.value,options:_,"onUpdate:value":t[2]||=e=>f.value=!1},null,8,[`value`])]),_:1})]),_:1},8,[`show`,`width`]),r(z(Pt),null,{default:V(()=>[r(z(Rt),{bordered:``,class:`app-header`},{default:V(()=>[j(`div`,Ln,[z(d)?(C(),W(z(ne),{key:0,quaternary:``,circle:``,class:`mobile-nav-trigger`,title:`打开导航`,"aria-label":`打开导航`,onClick:t[4]||=e=>f.value=!0},{icon:V(()=>[r(z(Le),null,{default:V(()=>[r(z(kn))]),_:1})]),_:1})):q(``,!0),j(`div`,Rn,[r(z(ke),{depth:`3`,class:`eyebrow`},{default:V(()=>[...t[10]||=[T(`KDAE CONTROL PLANE`,-1)]]),_:1}),j(`h1`,null,F(y.value),1)])]),j(`div`,zn,[r(z(lt),{round:``,size:`small`},{default:V(()=>[T(F(z(i).user?.username?.slice(0,1).toUpperCase()),1)]),_:1}),j(`div`,Bn,[j(`strong`,null,F(z(i).user?.username),1),t[11]||=j(`span`,null,`管理员`,-1)]),r(z(ne),{quaternary:``,circle:``,title:`退出登录`,onClick:x},{icon:V(()=>[r(z(Le),null,{default:V(()=>[r(z(Dn))]),_:1})]),_:1})])]),_:1}),r(z(Ft),{class:`app-content`,"content-style":`padding: var(--page-padding);`},{default:V(()=>[E.value?.check.updateAvailable&&!D.value?(C(),W(z(Re),{key:0,type:`info`,closable:``,class:`update-banner`,onClose:t[5]||=e=>D.value=!0},{default:V(()=>[j(`div`,Vn,[j(`span`,null,[t[15]||=T(` 面板有新版本 `,-1),j(`strong`,null,F(E.value.check.latest),1),T(`（当前 `+F(E.value.check.current)+`）。 `,1),E.value.status?.enabled&&E.value.status.updatable?(C(),N(P,{key:0},[T(`升级会替换面板二进制并重启自身，配置与账号数据都会保留。`)],64)):E.value.status&&!E.value.status.enabled?(C(),N(P,{key:1},[T(`可直接在这里启用一键升级，不需要 SSH。`)],64)):E.value.status?.problem?(C(),N(P,{key:2},[T(`当前无法一键升级：`+F(E.value.status.problem),1)],64)):z(a).isProcd?(C(),N(P,{key:3},[t[12]||=T(` 本部署由 opkg 升级：`,-1),t[13]||=j(`code`,{class:`mono`},`opkg update && opkg install kdae-panel luci-app-kdae-panel`,-1),t[14]||=T(`。 `,-1)],64)):(C(),N(P,{key:4},[T(`当前部署不支持一键升级，可重新执行一键部署命令。`)],64)),E.value.check.releasesUrl?(C(),N(`a`,{key:5,href:E.value.check.releasesUrl,target:`_blank`,rel:`noopener`},`查看发布说明`,8,Hn)):q(``,!0)]),r(Qe,{payload:E.value,label:`立即升级`},null,8,[`payload`])])]),_:1})):q(``,!0),r(n)]),_:1})]),_:1})]),_:1},8,[`has-sider`])}}});export{Un as default};