import{$t as e,A as t,An as n,Cn as r,Ct as i,Dt as a,En as o,Ft as s,Gn as c,Hn as l,In as u,Jt as d,Kn as f,Mt as p,Nt as m,On as h,Ot as g,Pn as _,Qt as v,Rn as y,Sn as b,T as x,Wn as S,Yt as C,Zt as w,_ as T,_n as E,bn as D,cr as O,en as k,f as A,fn as j,ft as M,gn as N,i as P,j as F,jt as ee,k as I,lr as te,nn as ne,nr as L,or as R,pt as z,tn as re,ur as B,vn as ie,wn as V,wt as H,x as ae,xn as oe,xt as se,y as ce,yn as le,yt as ue,zn as U,zt as de}from"./client-DzOxLNa2.js";import{r as fe,t as pe}from"./create-QAHl6yLj.js";import{t as me}from"./misc-DDs3MKLt.js";import{r as he}from"./light-B3Jb2Ihz.js";import{a as W,i as ge,n as _e,r as G,t as ve}from"./text-DPkxR-eM.js";import{n as ye,r as be,t as xe}from"./Dropdown-COZwuiAl.js";import{r as Se,t as Ce}from"./Icon-CBRp9WQI.js";import{t as we}from"./Alert-BQzzyFWo.js";import{S as Te,i as Ee,k as K,l as De,n as Oe,r as ke,t as Ae,x as je}from"./index-Cby6AgZX.js";import{t as Me}from"./CodeSlashOutline-BclWKJpx.js";import{t as Ne}from"./PanelUpdateAction-Cfj-kO4z.js";var Pe=V({name:`ChevronDownFilled`,render(){return o(`svg`,{viewBox:`0 0 16 16`,fill:`none`,xmlns:`http://www.w3.org/2000/svg`},o(`path`,{d:`M3.20041 5.73966C3.48226 5.43613 3.95681 5.41856 4.26034 5.70041L8 9.22652L11.7397 5.70041C12.0432 5.41856 12.5177 5.43613 12.7996 5.73966C13.0815 6.0432 13.0639 6.51775 12.7603 6.7996L8.51034 10.7996C8.22258 11.0668 7.77743 11.0668 7.48967 10.7996L3.23966 6.7996C2.93613 6.51775 2.91856 6.0432 3.20041 5.73966Z`,fill:`currentColor`}))}}),Fe=p&&`loading`in document.createElement(`img`);function Ie(e={}){let{root:t=null}=e;return{hash:`${e.rootMargin||`0px 0px 0px 0px`}-${Array.isArray(e.threshold)?e.threshold.join(`,`):e.threshold??`0`}`,options:Object.assign(Object.assign({},e),{root:(typeof t==`string`?document.querySelector(t):t)||document.documentElement})}}var q=new WeakMap,J=new WeakMap,Y=new WeakMap,Le=(e,t,n)=>{if(!e)return()=>{};let r=Ie(t),{root:i}=r.options,a,o=q.get(i);o?a=o:(a=new Map,q.set(i,a));let s,c;a.has(r.hash)?(c=a.get(r.hash),c[1].has(e)||(s=c[0],c[1].add(e),s.observe(e))):(s=new IntersectionObserver(e=>{e.forEach(e=>{if(e.isIntersecting){let t=J.get(e.target),n=Y.get(e.target);t&&t(),n&&(n.value=!0)}})},r.options),s.observe(e),c=[s,new Set([e])],a.set(r.hash,c));let l=!1,u=()=>{l||(J.delete(e),Y.delete(e),l=!0,c[1].has(e)&&(c[0].unobserve(e),c[1].delete(e)),c[1].size<=0&&a.delete(r.hash),a.size||q.delete(i))};return J.set(e,u),Y.set(e,n),u},Re=m(`n-avatar-group`),ze=C(`avatar`,`
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
`,[re(d(`&`,`--n-merged-color: var(--n-color-modal);`)),ne(d(`&`,`--n-merged-color: var(--n-color-popover);`)),d(`img`,`
 width: 100%;
 height: 100%;
 `),w(`text`,`
 white-space: nowrap;
 display: inline-block;
 position: absolute;
 left: 50%;
 top: 50%;
 `),C(`icon`,`
 vertical-align: bottom;
 font-size: calc(var(--n-merged-size) - 6px);
 `),w(`text`,`line-height: 1.25`)]),Be=V({name:`Avatar`,props:Object.assign(Object.assign({},F.props),{size:[String,Number],src:String,circle:{type:Boolean,default:void 0},objectFit:String,round:{type:Boolean,default:void 0},bordered:{type:Boolean,default:void 0},onError:Function,fallbackSrc:String,intersectionObserverOptions:Object,lazy:Boolean,onLoad:Function,renderPlaceholder:Function,renderFallback:Function,imgProps:Object,color:String}),slots:Object,setup(e){let{mergedClsPrefixRef:t,inlineThemeDisabled:n}=z(e),r=L(!1),i=null,o=L(null),s=L(null),l=()=>{let{value:e}=o;if(e&&(i===null||i!==e.innerHTML)){i=e.innerHTML;let{value:t}=s;if(t){let{offsetWidth:n,offsetHeight:r}=t,{offsetWidth:i,offsetHeight:a}=e,o=.9,s=Math.min(n/i*o,r/a*o,1);e.style.transform=`translateX(-50%) translateY(-50%) scale(${s})`}}},d=h(Re,null),f=N(()=>{let{size:t}=e;if(t)return t;let{size:n}=d||{};return n||`medium`}),p=F(`Avatar`,`-avatar`,ze,je,e,t),m=h(Se,null),g=N(()=>{if(d)return!0;let{round:t,circle:n}=e;return t!==void 0||n!==void 0?t||n:m?m.roundRef.value:!1}),v=N(()=>d?!0:e.bordered||!1),y=N(()=>{let t=f.value,n=g.value,r=v.value,{color:i}=e,{self:{borderRadius:a,fontSize:o,color:s,border:c,colorModal:l,colorPopover:u},common:{cubicBezierEaseInOut:d}}=p.value,m;return m=typeof t==`number`?`${t}px`:p.value.self[k(`height`,t)],{"--n-font-size":o,"--n-border":r?c:`none`,"--n-border-radius":n?`50%`:a,"--n-color":i||s,"--n-color-modal":i||l,"--n-color-popover":i||u,"--n-bezier":d,"--n-merged-size":`var(--n-avatar-size-override, ${m})`}}),b=n?M(`avatar`,N(()=>{let t=f.value,n=g.value,r=v.value,{color:i}=e,o=``;return t&&(typeof t==`number`?o+=`a${t}`:o+=t[0]),n&&(o+=`b`),r&&(o+=`c`),i&&(o+=a(i)),o}),y,e):void 0,x=L(!e.lazy);u(()=>{if(e.lazy&&e.intersectionObserverOptions){let t,n=c(()=>{t?.(),t=void 0,e.lazy&&(t=Le(s.value,e.intersectionObserverOptions,x))});_(()=>{n(),t?.()})}}),S(()=>e.src||e.imgProps?.src,()=>{r.value=!1});let C=L(!e.lazy);return{textRef:o,selfRef:s,mergedRoundRef:g,mergedClsPrefix:t,fitTextTransform:l,cssVars:n?void 0:y,themeClass:b?.themeClass,onRender:b?.onRender,hasLoadError:r,shouldStartLoading:x,loaded:C,mergedOnError:t=>{if(!x.value)return;r.value=!0;let{onError:n,imgProps:{onError:i}={}}=e;n?.(t),i?.(t)},mergedOnLoad:t=>{let{onLoad:n,imgProps:{onLoad:r}={}}=e;n?.(t),r?.(t),C.value=!0}}},render(){var e;let{$slots:t,src:n,mergedClsPrefix:r,lazy:i,onRender:a,loaded:s,hasLoadError:c,imgProps:l={}}=this;a?.();let u,d=!s&&!c&&(this.renderPlaceholder?this.renderPlaceholder():(e=this.$slots).placeholder?.call(e));return u=this.hasLoadError?this.renderFallback?this.renderFallback():ue(t.fallback,()=>[o(`img`,{src:this.fallbackSrc,style:{objectFit:this.objectFit}})]):se(t.default,e=>{if(e)return o(g,{onResize:this.fitTextTransform},{default:()=>o(`span`,{ref:`textRef`,class:`${r}-avatar__text`},e)});if(n||l.src){let e=this.src||l.src;return o(`img`,Object.assign(Object.assign({},l),{loading:Fe&&!this.intersectionObserverOptions&&i?`lazy`:`eager`,src:i&&this.intersectionObserverOptions?this.shouldStartLoading?e:void 0:e,"data-image-src":e,onLoad:this.mergedOnLoad,onError:this.mergedOnError,style:[l.style||``,{objectFit:this.objectFit},d?{height:`0`,width:`0`,visibility:`hidden`,position:`absolute`}:``]}))}}),o(`span`,{ref:`selfRef`,class:[`${r}-avatar`,this.themeClass],style:this.cssVars},u,i&&d)}});function Ve(e){let{baseColor:t,textColor2:n,bodyColor:r,cardColor:i,dividerColor:a,actionColor:o,scrollbarColor:s,scrollbarColorHover:c,invertedColor:l}=e;return{textColor:n,textColorInverted:`#FFF`,color:r,colorEmbedded:o,headerColor:i,headerColorInverted:l,footerColor:o,footerColorInverted:l,headerBorderColor:a,headerBorderColorInverted:l,footerBorderColor:a,footerBorderColorInverted:l,siderBorderColor:a,siderBorderColorInverted:l,siderColor:i,siderColorInverted:l,siderToggleButtonBorder:`1px solid ${a}`,siderToggleButtonColor:t,siderToggleButtonIconColor:n,siderToggleButtonIconColorInverted:n,siderToggleBarColor:de(r,s),siderToggleBarColorHover:de(r,c),__invertScrollbar:`true`}}var X=t({name:`Layout`,common:ae,peers:{Scrollbar:ce},self:Ve}),He=m(`n-layout-sider`),Z={type:String,default:`static`},Ue=C(`layout`,`
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
`,[C(`layout-scroll-container`,`
 overflow-x: hidden;
 box-sizing: border-box;
 height: 100%;
 `),v(`absolute-positioned`,`
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 `)]),We={embedded:Boolean,position:Z,nativeScrollbar:{type:Boolean,default:!0},scrollbarProps:Object,onScroll:Function,contentClass:String,contentStyle:{type:[String,Object],default:``},hasSider:Boolean,siderPlacement:{type:String,default:`left`}},Ge=m(`n-layout`);function Ke(e){return V({name:e?`LayoutContent`:`Layout`,props:Object.assign(Object.assign({},F.props),We),setup(e){let t=L(null),n=L(null),{mergedClsPrefixRef:r,inlineThemeDisabled:i}=z(e),a=F(`Layout`,`-layout`,Ue,X,e,r);function o(r,i){if(e.nativeScrollbar){let{value:e}=t;e&&(i===void 0?e.scrollTo(r):e.scrollTo(r,i))}else{let{value:e}=n;e&&e.scrollTo(r,i)}}U(Ge,e);let s=0,c=0,l=t=>{var n;let r=t.target;s=r.scrollLeft,c=r.scrollTop,(n=e.onScroll)==null||n.call(e,t)};ee(()=>{if(e.nativeScrollbar){let e=t.value;e&&(e.scrollTop=c,e.scrollLeft=s)}});let u={display:`flex`,flexWrap:`nowrap`,width:`100%`,flexDirection:`row`},d={scrollTo:o},f=N(()=>{let{common:{cubicBezierEaseInOut:t},self:n}=a.value;return{"--n-bezier":t,"--n-color":e.embedded?n.colorEmbedded:n.color,"--n-text-color":n.textColor}}),p=i?M(`layout`,N(()=>e.embedded?`e`:``),f,e):void 0;return Object.assign({mergedClsPrefix:r,scrollableElRef:t,scrollbarInstRef:n,hasSiderStyle:u,mergedTheme:a,handleNativeElScroll:l,cssVars:i?void 0:f,themeClass:p?.themeClass,onRender:p?.onRender},d)},render(){var t;let{mergedClsPrefix:n,hasSider:r}=this;(t=this.onRender)==null||t.call(this);let i=r?this.hasSiderStyle:void 0;return o(`div`,{class:[this.themeClass,e&&`${n}-layout-content`,`${n}-layout`,`${n}-layout--${this.position}-positioned`],style:this.cssVars},this.nativeScrollbar?o(`div`,{ref:`scrollableElRef`,class:[`${n}-layout-scroll-container`,this.contentClass],style:[this.contentStyle,i],onScroll:this.handleNativeElScroll},this.$slots):o(T,Object.assign({},this.scrollbarProps,{onScroll:this.onScroll,ref:`scrollbarInstRef`,theme:this.mergedTheme.peers.Scrollbar,themeOverrides:this.mergedTheme.peerOverrides.Scrollbar,contentClass:this.contentClass,contentStyle:[this.contentStyle,i]}),this.$slots))}})}var qe=Ke(!1),Je=Ke(!0),Ye=C(`layout-header`,`
 transition:
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 box-sizing: border-box;
 width: 100%;
 background-color: var(--n-color);
 color: var(--n-text-color);
`,[v(`absolute-positioned`,`
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 `),v(`bordered`,`
 border-bottom: solid 1px var(--n-border-color);
 `)]),Xe={position:Z,inverted:Boolean,bordered:{type:Boolean,default:!1}},Ze=V({name:`LayoutHeader`,props:Object.assign(Object.assign({},F.props),Xe),setup(e){let{mergedClsPrefixRef:t,inlineThemeDisabled:n}=z(e),r=F(`Layout`,`-layout-header`,Ye,X,e,t),i=N(()=>{let{common:{cubicBezierEaseInOut:t},self:n}=r.value,i={"--n-bezier":t};return e.inverted?(i[`--n-color`]=n.headerColorInverted,i[`--n-text-color`]=n.textColorInverted,i[`--n-border-color`]=n.headerBorderColorInverted):(i[`--n-color`]=n.headerColor,i[`--n-text-color`]=n.textColor,i[`--n-border-color`]=n.headerBorderColor),i}),a=n?M(`layout-header`,N(()=>e.inverted?`a`:`b`),i,e):void 0;return{mergedClsPrefix:t,cssVars:n?void 0:i,themeClass:a?.themeClass,onRender:a?.onRender}},render(){var e;let{mergedClsPrefix:t}=this;return(e=this.onRender)==null||e.call(this),o(`div`,{class:[`${t}-layout-header`,this.themeClass,this.position&&`${t}-layout-header--${this.position}-positioned`,this.bordered&&`${t}-layout-header--bordered`],style:this.cssVars},this.$slots)}}),Qe=C(`layout-sider`,`
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
`,[v(`bordered`,[w(`border`,`
 content: "";
 position: absolute;
 top: 0;
 bottom: 0;
 width: 1px;
 background-color: var(--n-border-color);
 transition: background-color .3s var(--n-bezier);
 `)]),w(`left-placement`,[v(`bordered`,[w(`border`,`
 right: 0;
 `)])]),v(`right-placement`,`
 justify-content: flex-start;
 `,[v(`bordered`,[w(`border`,`
 left: 0;
 `)]),v(`collapsed`,[C(`layout-toggle-button`,[C(`base-icon`,`
 transform: rotate(180deg);
 `)]),C(`layout-toggle-bar`,[d(`&:hover`,[w(`top`,{transform:`rotate(-12deg) scale(1.15) translateY(-2px)`}),w(`bottom`,{transform:`rotate(12deg) scale(1.15) translateY(2px)`})])])]),C(`layout-toggle-button`,`
 left: 0;
 transform: translateX(-50%) translateY(-50%);
 `,[C(`base-icon`,`
 transform: rotate(0);
 `)]),C(`layout-toggle-bar`,`
 left: -28px;
 transform: rotate(180deg);
 `,[d(`&:hover`,[w(`top`,{transform:`rotate(12deg) scale(1.15) translateY(-2px)`}),w(`bottom`,{transform:`rotate(-12deg) scale(1.15) translateY(2px)`})])])]),v(`collapsed`,[C(`layout-toggle-bar`,[d(`&:hover`,[w(`top`,{transform:`rotate(-12deg) scale(1.15) translateY(-2px)`}),w(`bottom`,{transform:`rotate(12deg) scale(1.15) translateY(2px)`})])]),C(`layout-toggle-button`,[C(`base-icon`,`
 transform: rotate(0);
 `)])]),C(`layout-toggle-button`,`
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
 `,[C(`base-icon`,`
 transition: transform .3s var(--n-bezier);
 transform: rotate(180deg);
 `)]),C(`layout-toggle-bar`,`
 cursor: pointer;
 height: 72px;
 width: 32px;
 position: absolute;
 top: calc(50% - 36px);
 right: -28px;
 `,[w(`top, bottom`,`
 position: absolute;
 width: 4px;
 border-radius: 2px;
 height: 38px;
 left: 14px;
 transition: 
 background-color .3s var(--n-bezier),
 transform .3s var(--n-bezier);
 `),w(`bottom`,`
 position: absolute;
 top: 34px;
 `),d(`&:hover`,[w(`top`,{transform:`rotate(12deg) scale(1.15) translateY(-2px)`}),w(`bottom`,{transform:`rotate(-12deg) scale(1.15) translateY(2px)`})]),w(`top, bottom`,{backgroundColor:`var(--n-toggle-bar-color)`}),d(`&:hover`,[w(`top, bottom`,{backgroundColor:`var(--n-toggle-bar-color-hover)`})])]),w(`border`,`
 position: absolute;
 top: 0;
 right: 0;
 bottom: 0;
 width: 1px;
 transition: background-color .3s var(--n-bezier);
 `),C(`layout-sider-scroll-container`,`
 flex-grow: 1;
 flex-shrink: 0;
 box-sizing: border-box;
 height: 100%;
 opacity: 0;
 transition: opacity .3s var(--n-bezier);
 max-width: 100%;
 `),v(`show-content`,[C(`layout-sider-scroll-container`,{opacity:1})]),v(`absolute-positioned`,`
 position: absolute;
 left: 0;
 top: 0;
 bottom: 0;
 `)]),$e=V({props:{clsPrefix:{type:String,required:!0},onClick:Function},render(){let{clsPrefix:e}=this;return o(`div`,{onClick:this.onClick,class:`${e}-layout-toggle-bar`},o(`div`,{class:`${e}-layout-toggle-bar__top`}),o(`div`,{class:`${e}-layout-toggle-bar__bottom`}))}}),et=V({name:`LayoutToggleButton`,props:{clsPrefix:{type:String,required:!0},onClick:Function},render(){let{clsPrefix:e}=this;return o(`div`,{class:`${e}-layout-toggle-button`,onClick:this.onClick},o(I,{clsPrefix:e},{default:()=>o(be,null)}))}}),tt={position:Z,bordered:Boolean,collapsedWidth:{type:Number,default:48},width:{type:[Number,String],default:272},contentClass:String,contentStyle:{type:[String,Object],default:``},collapseMode:{type:String,default:`transform`},collapsed:{type:Boolean,default:void 0},defaultCollapsed:Boolean,showCollapsedContent:{type:Boolean,default:!0},showTrigger:{type:[Boolean,String],default:!1},nativeScrollbar:{type:Boolean,default:!0},inverted:Boolean,scrollbarProps:Object,triggerClass:String,triggerStyle:[String,Object],collapsedTriggerClass:String,collapsedTriggerStyle:[String,Object],"onUpdate:collapsed":[Function,Array],onUpdateCollapsed:[Function,Array],onAfterEnter:Function,onAfterLeave:Function,onExpand:[Function,Array],onCollapse:[Function,Array],onScroll:Function},nt=V({name:`LayoutSider`,props:Object.assign(Object.assign({},F.props),tt),setup(e){let t=h(Ge),n=L(null),r=L(null),i=L(e.defaultCollapsed),a=W(R(e,`collapsed`),i),o=N(()=>G(a.value?e.collapsedWidth:e.width)),s=N(()=>e.collapseMode===`transform`?{minWidth:G(e.width)}:{}),c=N(()=>t?t.siderPlacement:`left`);function l(t,i){if(e.nativeScrollbar){let{value:e}=n;e&&(i===void 0?e.scrollTo(t):e.scrollTo(t,i))}else{let{value:e}=r;e&&e.scrollTo(t,i)}}function u(){let{"onUpdate:collapsed":t,onUpdateCollapsed:n,onExpand:r,onCollapse:o}=e,{value:s}=a;n&&H(n,!s),t&&H(t,!s),i.value=!s,s?r&&H(r):o&&H(o)}let d=0,f=0,p=t=>{var n;let r=t.target;d=r.scrollLeft,f=r.scrollTop,(n=e.onScroll)==null||n.call(e,t)};ee(()=>{if(e.nativeScrollbar){let e=n.value;e&&(e.scrollTop=f,e.scrollLeft=d)}}),U(He,{collapsedRef:a,collapseModeRef:R(e,`collapseMode`)});let{mergedClsPrefixRef:m,inlineThemeDisabled:g}=z(e),_=F(`Layout`,`-layout-sider`,Qe,X,e,m);function v(t){var n,r;t.propertyName===`max-width`&&(a.value?(n=e.onAfterLeave)==null||n.call(e):(r=e.onAfterEnter)==null||r.call(e))}let y={scrollTo:l},b=N(()=>{let{common:{cubicBezierEaseInOut:t},self:n}=_.value,{siderToggleButtonColor:r,siderToggleButtonBorder:i,siderToggleBarColor:a,siderToggleBarColorHover:o}=n,s={"--n-bezier":t,"--n-toggle-button-color":r,"--n-toggle-button-border":i,"--n-toggle-bar-color":a,"--n-toggle-bar-color-hover":o};return e.inverted?(s[`--n-color`]=n.siderColorInverted,s[`--n-text-color`]=n.textColorInverted,s[`--n-border-color`]=n.siderBorderColorInverted,s[`--n-toggle-button-icon-color`]=n.siderToggleButtonIconColorInverted,s.__invertScrollbar=n.__invertScrollbar):(s[`--n-color`]=n.siderColor,s[`--n-text-color`]=n.textColor,s[`--n-border-color`]=n.siderBorderColor,s[`--n-toggle-button-icon-color`]=n.siderToggleButtonIconColor),s}),x=g?M(`layout-sider`,N(()=>e.inverted?`a`:`b`),b,e):void 0;return Object.assign({scrollableElRef:n,scrollbarInstRef:r,mergedClsPrefix:m,mergedTheme:_,styleMaxWidth:o,mergedCollapsed:a,scrollContainerStyle:s,siderPlacement:c,handleNativeElScroll:p,handleTransitionend:v,handleTriggerClick:u,inlineThemeDisabled:g,cssVars:b,themeClass:x?.themeClass,onRender:x?.onRender},y)},render(){var e;let{mergedClsPrefix:t,mergedCollapsed:n,showTrigger:r}=this;return(e=this.onRender)==null||e.call(this),o(`aside`,{class:[`${t}-layout-sider`,this.themeClass,`${t}-layout-sider--${this.position}-positioned`,`${t}-layout-sider--${this.siderPlacement}-placement`,this.bordered&&`${t}-layout-sider--bordered`,n&&`${t}-layout-sider--collapsed`,(!n||this.showCollapsedContent)&&`${t}-layout-sider--show-content`],onTransitionend:this.handleTransitionend,style:[this.inlineThemeDisabled?void 0:this.cssVars,{maxWidth:this.styleMaxWidth,width:G(this.width)}]},this.nativeScrollbar?o(`div`,{class:[`${t}-layout-sider-scroll-container`,this.contentClass],onScroll:this.handleNativeElScroll,style:[this.scrollContainerStyle,{overflow:`auto`},this.contentStyle],ref:`scrollableElRef`},this.$slots):o(T,Object.assign({},this.scrollbarProps,{onScroll:this.onScroll,ref:`scrollbarInstRef`,style:this.scrollContainerStyle,contentStyle:this.contentStyle,contentClass:this.contentClass,theme:this.mergedTheme.peers.Scrollbar,themeOverrides:this.mergedTheme.peerOverrides.Scrollbar,builtinThemeOverrides:this.inverted&&this.cssVars.__invertScrollbar===`true`?{colorHover:`rgba(255, 255, 255, .4)`,color:`rgba(255, 255, 255, .3)`}:void 0}),this.$slots),r?o(r===`bar`?$e:et,{clsPrefix:t,class:n?this.collapsedTriggerClass:this.triggerClass,style:n?this.collapsedTriggerStyle:this.triggerStyle,onClick:this.handleTriggerClick}):null,this.bordered?o(`div`,{class:`${t}-layout-sider__border`}):null)}}),Q=m(`n-menu`),rt=m(`n-submenu`),it=m(`n-menu-item-group`),at=[d(`&::before`,`background-color: var(--n-item-color-hover);`),w(`arrow`,`
 color: var(--n-arrow-color-hover);
 `),w(`icon`,`
 color: var(--n-item-icon-color-hover);
 `),C(`menu-item-content-header`,`
 color: var(--n-item-text-color-hover);
 `,[d(`a`,`
 color: var(--n-item-text-color-hover);
 `),w(`extra`,`
 color: var(--n-item-text-color-hover);
 `)])],ot=[w(`icon`,`
 color: var(--n-item-icon-color-hover-horizontal);
 `),C(`menu-item-content-header`,`
 color: var(--n-item-text-color-hover-horizontal);
 `,[d(`a`,`
 color: var(--n-item-text-color-hover-horizontal);
 `),w(`extra`,`
 color: var(--n-item-text-color-hover-horizontal);
 `)])],st=d([C(`menu`,`
 background-color: var(--n-color);
 color: var(--n-item-text-color);
 overflow: hidden;
 transition: background-color .3s var(--n-bezier);
 box-sizing: border-box;
 font-size: var(--n-font-size);
 padding-bottom: 6px;
 `,[v(`horizontal`,`
 max-width: 100%;
 width: 100%;
 display: flex;
 overflow: hidden;
 padding-bottom: 0;
 `,[C(`submenu`,`margin: 0;`),C(`menu-item`,`margin: 0;`),C(`menu-item-content`,`
 padding: 0 20px;
 border-bottom: 2px solid #0000;
 `,[d(`&::before`,`display: none;`),v(`selected`,`border-bottom: 2px solid var(--n-border-color-horizontal)`)]),C(`menu-item-content`,[v(`selected`,[w(`icon`,`color: var(--n-item-icon-color-active-horizontal);`),C(`menu-item-content-header`,`
 color: var(--n-item-text-color-active-horizontal);
 `,[d(`a`,`color: var(--n-item-text-color-active-horizontal);`),w(`extra`,`color: var(--n-item-text-color-active-horizontal);`)])]),v(`child-active`,`
 border-bottom: 2px solid var(--n-border-color-horizontal);
 `,[C(`menu-item-content-header`,`
 color: var(--n-item-text-color-child-active-horizontal);
 `,[d(`a`,`
 color: var(--n-item-text-color-child-active-horizontal);
 `),w(`extra`,`
 color: var(--n-item-text-color-child-active-horizontal);
 `)]),w(`icon`,`
 color: var(--n-item-icon-color-child-active-horizontal);
 `)]),e(`disabled`,[e(`selected, child-active`,[d(`&:focus-within`,ot)]),v(`selected`,[$(null,[w(`icon`,`color: var(--n-item-icon-color-active-hover-horizontal);`),C(`menu-item-content-header`,`
 color: var(--n-item-text-color-active-hover-horizontal);
 `,[d(`a`,`color: var(--n-item-text-color-active-hover-horizontal);`),w(`extra`,`color: var(--n-item-text-color-active-hover-horizontal);`)])])]),v(`child-active`,[$(null,[w(`icon`,`color: var(--n-item-icon-color-child-active-hover-horizontal);`),C(`menu-item-content-header`,`
 color: var(--n-item-text-color-child-active-hover-horizontal);
 `,[d(`a`,`color: var(--n-item-text-color-child-active-hover-horizontal);`),w(`extra`,`color: var(--n-item-text-color-child-active-hover-horizontal);`)])])]),$(`border-bottom: 2px solid var(--n-border-color-horizontal);`,ot)]),C(`menu-item-content-header`,[d(`a`,`color: var(--n-item-text-color-horizontal);`)])])]),e(`responsive`,[C(`menu-item-content-header`,`
 overflow: hidden;
 text-overflow: ellipsis;
 `)]),v(`collapsed`,[C(`menu-item-content`,[v(`selected`,[d(`&::before`,`
 background-color: var(--n-item-color-active-collapsed) !important;
 `)]),C(`menu-item-content-header`,`opacity: 0;`),w(`arrow`,`opacity: 0;`),w(`icon`,`color: var(--n-item-icon-color-collapsed);`)])]),C(`menu-item`,`
 height: var(--n-item-height);
 margin-top: 6px;
 position: relative;
 `),C(`menu-item-content`,`
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
 `,[d(`> *`,`z-index: 1;`),d(`&::before`,`
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
 `),v(`disabled`,`
 opacity: .45;
 cursor: not-allowed;
 `),v(`collapsed`,[w(`arrow`,`transform: rotate(0);`)]),v(`selected`,[d(`&::before`,`background-color: var(--n-item-color-active);`),w(`arrow`,`color: var(--n-arrow-color-active);`),w(`icon`,`color: var(--n-item-icon-color-active);`),C(`menu-item-content-header`,`
 color: var(--n-item-text-color-active);
 `,[d(`a`,`color: var(--n-item-text-color-active);`),w(`extra`,`color: var(--n-item-text-color-active);`)])]),v(`child-active`,[C(`menu-item-content-header`,`
 color: var(--n-item-text-color-child-active);
 `,[d(`a`,`
 color: var(--n-item-text-color-child-active);
 `),w(`extra`,`
 color: var(--n-item-text-color-child-active);
 `)]),w(`arrow`,`
 color: var(--n-arrow-color-child-active);
 `),w(`icon`,`
 color: var(--n-item-icon-color-child-active);
 `)]),e(`disabled`,[e(`selected, child-active`,[d(`&:focus-within`,at)]),v(`selected`,[$(null,[w(`arrow`,`color: var(--n-arrow-color-active-hover);`),w(`icon`,`color: var(--n-item-icon-color-active-hover);`),C(`menu-item-content-header`,`
 color: var(--n-item-text-color-active-hover);
 `,[d(`a`,`color: var(--n-item-text-color-active-hover);`),w(`extra`,`color: var(--n-item-text-color-active-hover);`)])])]),v(`child-active`,[$(null,[w(`arrow`,`color: var(--n-arrow-color-child-active-hover);`),w(`icon`,`color: var(--n-item-icon-color-child-active-hover);`),C(`menu-item-content-header`,`
 color: var(--n-item-text-color-child-active-hover);
 `,[d(`a`,`color: var(--n-item-text-color-child-active-hover);`),w(`extra`,`color: var(--n-item-text-color-child-active-hover);`)])])]),v(`selected`,[$(null,[d(`&::before`,`background-color: var(--n-item-color-active-hover);`)])]),$(null,at)]),w(`icon`,`
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
 `),w(`arrow`,`
 grid-area: arrow;
 font-size: 16px;
 color: var(--n-arrow-color);
 transform: rotate(180deg);
 opacity: 1;
 transition:
 color .3s var(--n-bezier),
 transform 0.2s var(--n-bezier),
 opacity 0.2s var(--n-bezier);
 `),C(`menu-item-content-header`,`
 grid-area: content;
 transition:
 color .3s var(--n-bezier),
 opacity .3s var(--n-bezier);
 opacity: 1;
 white-space: nowrap;
 color: var(--n-item-text-color);
 `,[d(`a`,`
 outline: none;
 text-decoration: none;
 transition: color .3s var(--n-bezier);
 color: var(--n-item-text-color);
 `,[d(`&::before`,`
 content: "";
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 `)]),w(`extra`,`
 font-size: .93em;
 color: var(--n-group-text-color);
 transition: color .3s var(--n-bezier);
 `)])]),C(`submenu`,`
 cursor: pointer;
 position: relative;
 margin-top: 6px;
 `,[C(`menu-item-content`,`
 height: var(--n-item-height);
 `),C(`submenu-children`,`
 overflow: hidden;
 padding: 0;
 `,[Te({duration:`.2s`})])]),C(`menu-item-group`,[C(`menu-item-group-title`,`
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
 `)])]),C(`menu-tooltip`,[d(`a`,`
 color: inherit;
 text-decoration: none;
 `)]),C(`menu-divider`,`
 transition: background-color .3s var(--n-bezier);
 background-color: var(--n-divider-color);
 height: 1px;
 margin: 6px 18px;
 `)]);function $(e,t){return[v(`hover`,e,t),d(`&:hover`,e,t)]}var ct=V({name:`MenuOptionContent`,props:{collapsed:Boolean,disabled:Boolean,title:[String,Function],icon:Function,extra:[String,Function],showArrow:Boolean,childActive:Boolean,hover:Boolean,paddingLeft:Number,selected:Boolean,maxIconSize:{type:Number,required:!0},activeIconSize:{type:Number,required:!0},iconMarginRight:{type:Number,required:!0},clsPrefix:{type:String,required:!0},onClick:Function,tmNode:{type:Object,required:!0},isEllipsisPlaceholder:Boolean},setup(e){let{props:t}=h(Q);return{menuProps:t,style:N(()=>{let{paddingLeft:t}=e;return{paddingLeft:t&&`${t}px`}}),iconStyle:N(()=>{let{maxIconSize:t,activeIconSize:n,iconMarginRight:r}=e;return{width:`${t}px`,height:`${t}px`,fontSize:`${n}px`,marginRight:`${r}px`}})}},render(){let{clsPrefix:e,tmNode:t,menuProps:{renderIcon:n,renderLabel:r,renderExtra:i,expandIcon:a}}=this,s=n?n(t.rawNode):K(this.icon);return o(`div`,{onClick:e=>{var t;(t=this.onClick)==null||t.call(this,e)},role:`none`,class:[`${e}-menu-item-content`,{[`${e}-menu-item-content--selected`]:this.selected,[`${e}-menu-item-content--collapsed`]:this.collapsed,[`${e}-menu-item-content--child-active`]:this.childActive,[`${e}-menu-item-content--disabled`]:this.disabled,[`${e}-menu-item-content--hover`]:this.hover}],style:this.style},s&&o(`div`,{class:`${e}-menu-item-content__icon`,style:this.iconStyle,role:`none`},[s]),o(`div`,{class:`${e}-menu-item-content-header`,role:`none`},this.isEllipsisPlaceholder?this.title:r?r(t.rawNode):K(this.title),this.extra||i?o(`span`,{class:`${e}-menu-item-content-header__extra`},` `,i?i(t.rawNode):K(this.extra)):null),this.showArrow?o(I,{ariaHidden:!0,class:`${e}-menu-item-content__arrow`,clsPrefix:e},{default:()=>a?a(t.rawNode):o(Pe,null)}):null)}}),lt=8;function ut(e){let t=h(Q),{props:n,mergedCollapsedRef:r}=t,i=h(rt,null),a=h(it,null),o=N(()=>n.mode===`horizontal`),s=N(()=>o.value?n.dropdownPlacement:`tmNodes`in e?`right-start`:`right`),c=N(()=>Math.max(n.collapsedIconSize??n.iconSize,n.iconSize));return{dropdownPlacement:s,activeIconSize:N(()=>!o.value&&e.root&&r.value?n.collapsedIconSize??n.iconSize:n.iconSize),maxIconSize:c,paddingLeft:N(()=>{if(o.value)return;let{collapsedWidth:t,indent:s,rootIndent:l}=n,{root:u,isGroup:d}=e,f=l===void 0?s:l;return u?r.value?t/2-c.value/2:f:a&&typeof a.paddingLeftRef.value==`number`?s/2+a.paddingLeftRef.value:i&&typeof i.paddingLeftRef.value==`number`?(d?s/2:s)+i.paddingLeftRef.value:0}),iconMarginRight:N(()=>{let{collapsedWidth:t,indent:i,rootIndent:a}=n,{value:s}=c,{root:l}=e;return o.value||!l||!r.value?lt:(a===void 0?i:a)+s+lt-(t+s)/2}),NMenu:t,NSubmenu:i,NMenuOptionGroup:a}}var dt={internalKey:{type:[String,Number],required:!0},root:Boolean,isGroup:Boolean,level:{type:Number,required:!0},title:[String,Function],extra:[String,Function]},ft=V({name:`MenuDivider`,setup(){let{mergedClsPrefixRef:e,isHorizontalRef:t}=h(Q);return()=>t.value?null:o(`div`,{class:`${e.value}-menu-divider`})}}),pt=Object.assign(Object.assign({},dt),{tmNode:{type:Object,required:!0},disabled:Boolean,icon:Function,onClick:Function}),mt=i(pt),ht=V({name:`MenuOption`,props:pt,setup(e){let t=ut(e),{NSubmenu:n,NMenu:r,NMenuOptionGroup:i}=t,{props:a,mergedClsPrefixRef:o,mergedCollapsedRef:c}=r,l=n?n.mergedDisabledRef:i?i.mergedDisabledRef:{value:!1},u=N(()=>l.value||e.disabled);function d(t){let{onClick:n}=e;n&&n(t)}function f(t){u.value||(r.doSelect(e.internalKey,e.tmNode.rawNode),d(t))}return{mergedClsPrefix:o,dropdownPlacement:t.dropdownPlacement,paddingLeft:t.paddingLeft,iconMarginRight:t.iconMarginRight,maxIconSize:t.maxIconSize,activeIconSize:t.activeIconSize,mergedTheme:r.mergedThemeRef,menuProps:a,dropdownEnabled:s(()=>e.root&&c.value&&a.mode!==`horizontal`&&!u.value),selected:s(()=>r.mergedValueRef.value===e.internalKey),mergedDisabled:u,handleClick:f}},render(){let{mergedClsPrefix:e,mergedTheme:t,tmNode:n,menuProps:{renderLabel:r,nodeProps:i}}=this,a=i?.(n.rawNode);return o(`div`,Object.assign({},a,{role:`menuitem`,class:[`${e}-menu-item`,a?.class]}),o(ye,{theme:t.peers.Tooltip,themeOverrides:t.peerOverrides.Tooltip,trigger:`hover`,placement:this.dropdownPlacement,disabled:!this.dropdownEnabled||this.title===void 0,internalExtraClass:[`menu-tooltip`]},{default:()=>r?r(n.rawNode):K(this.title),trigger:()=>o(ct,{tmNode:n,clsPrefix:e,paddingLeft:this.paddingLeft,iconMarginRight:this.iconMarginRight,maxIconSize:this.maxIconSize,activeIconSize:this.activeIconSize,selected:this.selected,title:this.title,extra:this.extra,disabled:this.mergedDisabled,icon:this.icon,onClick:this.handleClick})}))}}),gt=Object.assign(Object.assign({},dt),{tmNode:{type:Object,required:!0},tmNodes:{type:Array,required:!0}}),_t=i(gt),vt=V({name:`MenuOptionGroup`,props:gt,setup(e){let t=ut(e),{NSubmenu:n}=t,r=N(()=>n?.mergedDisabledRef.value?!0:e.tmNode.disabled);U(it,{paddingLeftRef:t.paddingLeft,mergedDisabledRef:r});let{mergedClsPrefixRef:i,props:a}=h(Q);return function(){let{value:n}=i,r=t.paddingLeft.value,{nodeProps:s}=a,c=s?.(e.tmNode.rawNode);return o(`div`,{class:`${n}-menu-item-group`,role:`group`},o(`div`,Object.assign({},c,{class:[`${n}-menu-item-group-title`,c?.class],style:[c?.style||``,r===void 0?``:`padding-left: ${r}px;`]}),K(e.title),e.extra?o(j,null,` `,K(e.extra)):null),o(`div`,null,e.tmNodes.map(e=>xt(e,a))))}}});function yt(e){return e.type===`divider`||e.type===`render`}function bt(e){return e.type===`divider`}function xt(e,t){let{rawNode:n}=e,{show:r}=n;if(r===!1)return null;if(yt(n))return bt(n)?o(ft,Object.assign({key:e.key},n.props)):null;let{labelField:i}=t,{key:a,level:s,isGroup:c}=e,l=Object.assign(Object.assign({},n),{title:n.title||n[i],extra:n.titleExtra||n.extra,key:a,internalKey:a,level:s,root:s===0,isGroup:c});return e.children?e.isGroup?o(vt,he(l,_t,{tmNode:e,tmNodes:e.children,key:a})):o(wt,he(l,Ct,{key:a,rawNodes:n[t.childrenField],tmNodes:e.children,tmNode:e})):o(ht,he(l,mt,{key:a,tmNode:e}))}var St=Object.assign(Object.assign({},dt),{rawNodes:{type:Array,default:()=>[]},tmNodes:{type:Array,default:()=>[]},tmNode:{type:Object,required:!0},disabled:Boolean,icon:Function,onClick:Function,domId:String,virtualChildActive:{type:Boolean,default:void 0},isEllipsisPlaceholder:Boolean}),Ct=i(St),wt=V({name:`Submenu`,props:St,setup(e){let t=ut(e),{NMenu:n,NSubmenu:r}=t,{props:i,mergedCollapsedRef:a,mergedThemeRef:o}=n,c=N(()=>{let{disabled:t}=e;return r?.mergedDisabledRef.value||i.disabled?!0:t}),l=L(!1);U(rt,{paddingLeftRef:t.paddingLeft,mergedDisabledRef:c}),U(it,null);function u(){let{onClick:t}=e;t&&t()}function d(){c.value||(a.value||n.toggleExpand(e.internalKey),u())}function f(e){l.value=e}return{menuProps:i,mergedTheme:o,doSelect:n.doSelect,inverted:n.invertedRef,isHorizontal:n.isHorizontalRef,mergedClsPrefix:n.mergedClsPrefixRef,maxIconSize:t.maxIconSize,activeIconSize:t.activeIconSize,iconMarginRight:t.iconMarginRight,dropdownPlacement:t.dropdownPlacement,dropdownShow:l,paddingLeft:t.paddingLeft,mergedDisabled:c,mergedValue:n.mergedValueRef,childActive:s(()=>e.virtualChildActive??n.activePathRef.value.includes(e.internalKey)),collapsed:N(()=>i.mode===`horizontal`?!1:a.value?!0:!n.mergedExpandedKeysRef.value.includes(e.internalKey)),dropdownEnabled:N(()=>!c.value&&(i.mode===`horizontal`||a.value)),handlePopoverShowChange:f,handleClick:d}},render(){let{mergedClsPrefix:e,menuProps:{renderIcon:t,renderLabel:n}}=this,r=()=>{let{isHorizontal:e,paddingLeft:t,collapsed:n,mergedDisabled:r,maxIconSize:i,activeIconSize:a,title:s,childActive:c,icon:l,handleClick:u,menuProps:{nodeProps:d},dropdownShow:f,iconMarginRight:p,tmNode:m,mergedClsPrefix:h,isEllipsisPlaceholder:g,extra:_}=this,v=d?.(m.rawNode);return o(`div`,Object.assign({},v,{class:[`${h}-menu-item`,v?.class],role:`menuitem`}),o(ct,{tmNode:m,paddingLeft:t,collapsed:n,disabled:r,iconMarginRight:p,maxIconSize:i,activeIconSize:a,title:s,extra:_,showArrow:!e,childActive:c,clsPrefix:h,icon:l,hover:f,onClick:u,isEllipsisPlaceholder:g}))},i=()=>o(x,null,{default:()=>{let{tmNodes:t,collapsed:n}=this;return n?null:o(`div`,{class:`${e}-submenu-children`,role:`menu`},t.map(e=>xt(e,this.menuProps)))}});return this.root?o(xe,Object.assign({size:`large`,trigger:`hover`},this.menuProps?.dropdownProps,{themeOverrides:this.mergedTheme.peerOverrides.Dropdown,theme:this.mergedTheme.peers.Dropdown,builtinThemeOverrides:{fontSizeLarge:`14px`,optionIconSizeLarge:`18px`},value:this.mergedValue,disabled:!this.dropdownEnabled,placement:this.dropdownPlacement,keyField:this.menuProps.keyField,labelField:this.menuProps.labelField,childrenField:this.menuProps.childrenField,onUpdateShow:this.handlePopoverShowChange,options:this.rawNodes,onSelect:this.doSelect,inverted:this.inverted,renderIcon:t,renderLabel:n}),{default:()=>o(`div`,{class:`${e}-submenu`,role:`menu`,"aria-expanded":!this.collapsed,id:this.domId},r(),this.isHorizontal?null:i())}):o(`div`,{class:`${e}-submenu`,role:`menu`,"aria-expanded":!this.collapsed,id:this.domId},r(),i())}}),Tt=V({name:`Menu`,inheritAttrs:!1,props:Object.assign(Object.assign({},F.props),{options:{type:Array,default:()=>[]},collapsed:{type:Boolean,default:void 0},collapsedWidth:{type:Number,default:48},iconSize:{type:Number,default:20},collapsedIconSize:{type:Number,default:24},rootIndent:Number,indent:{type:Number,default:32},labelField:{type:String,default:`label`},keyField:{type:String,default:`key`},childrenField:{type:String,default:`children`},disabledField:{type:String,default:`disabled`},defaultExpandAll:Boolean,defaultExpandedKeys:Array,expandedKeys:Array,value:[String,Number],defaultValue:{type:[String,Number],default:null},mode:{type:String,default:`vertical`},watchProps:{type:Array,default:void 0},disabled:Boolean,show:{type:Boolean,default:!0},inverted:Boolean,"onUpdate:expandedKeys":[Function,Array],onUpdateExpandedKeys:[Function,Array],onUpdateValue:[Function,Array],"onUpdate:value":[Function,Array],expandIcon:Function,renderIcon:Function,renderLabel:Function,renderExtra:Function,dropdownProps:Object,accordion:Boolean,nodeProps:Function,dropdownPlacement:{type:String,default:`bottom`},responsive:Boolean,items:Array,onOpenNamesChange:[Function,Array],onSelect:[Function,Array],onExpandedNamesChange:[Function,Array],expandedNames:Array,defaultExpandedNames:Array}),setup(e){let{mergedClsPrefixRef:t,inlineThemeDisabled:n}=z(e),r=F(`Menu`,`-menu`,st,De,e,t),i=h(He,null),a=N(()=>{let{collapsed:t}=e;if(t!==void 0)return t;if(i){let{collapseModeRef:e,collapsedRef:t}=i;if(e.value===`width`)return t.value??!1}return!1}),s=N(()=>{let{keyField:t,childrenField:n,disabledField:r}=e;return pe(e.items||e.options,{getIgnored(e){return yt(e)},getChildren(e){return e[n]},getDisabled(e){return e[r]},getKey(e){return e[t]??e.name}})}),l=N(()=>new Set(s.value.treeNodes.map(e=>e.key))),{watchProps:u}=e,d=L(null);u?.includes(`defaultValue`)?c(()=>{d.value=e.defaultValue}):d.value=e.defaultValue;let f=W(R(e,`value`),d),p=L([]),m=()=>{p.value=e.defaultExpandAll?s.value.getNonLeafKeys():e.defaultExpandedNames||e.defaultExpandedKeys||s.value.getPath(f.value,{includeSelf:!1}).keyPath};u?.includes(`defaultExpandedKeys`)?c(m):m();let g=ge(e,[`expandedNames`,`expandedKeys`]),_=W(g,p),v=N(()=>s.value.treeNodes),y=N(()=>s.value.getPath(f.value).keyPath);U(Q,{props:e,mergedCollapsedRef:a,mergedThemeRef:r,mergedValueRef:f,mergedExpandedKeysRef:_,activePathRef:y,mergedClsPrefixRef:t,isHorizontalRef:N(()=>e.mode===`horizontal`),invertedRef:R(e,`inverted`),doSelect:b,toggleExpand:S});function b(t,n){let{"onUpdate:value":r,onUpdateValue:i,onSelect:a}=e;i&&H(i,t,n),r&&H(r,t,n),a&&H(a,t,n),d.value=t}function x(t){let{"onUpdate:expandedKeys":n,onUpdateExpandedKeys:r,onExpandedNamesChange:i,onOpenNamesChange:a}=e;n&&H(n,t),r&&H(r,t),i&&H(i,t),a&&H(a,t),p.value=t}function S(t){let n=Array.from(_.value),r=n.findIndex(e=>e===t);if(~r)n.splice(r,1);else{if(e.accordion&&l.value.has(t)){let e=n.findIndex(e=>l.value.has(e));e>-1&&n.splice(e,1)}n.push(t)}x(n)}let C=t=>{let n=s.value.getPath(t??f.value,{includeSelf:!1}).keyPath;if(!n.length)return;let r=Array.from(_.value),i=new Set([...r,...n]);e.accordion&&l.value.forEach(e=>{i.has(e)&&!n.includes(e)&&i.delete(e)}),x(Array.from(i))},w=N(()=>{let{inverted:t}=e,{common:{cubicBezierEaseInOut:n},self:i}=r.value,{borderRadius:a,borderColorHorizontal:o,fontSize:s,itemHeight:c,dividerColor:l}=i,u={"--n-divider-color":l,"--n-bezier":n,"--n-font-size":s,"--n-border-color-horizontal":o,"--n-border-radius":a,"--n-item-height":c};return t?(u[`--n-group-text-color`]=i.groupTextColorInverted,u[`--n-color`]=i.colorInverted,u[`--n-item-text-color`]=i.itemTextColorInverted,u[`--n-item-text-color-hover`]=i.itemTextColorHoverInverted,u[`--n-item-text-color-active`]=i.itemTextColorActiveInverted,u[`--n-item-text-color-child-active`]=i.itemTextColorChildActiveInverted,u[`--n-item-text-color-child-active-hover`]=i.itemTextColorChildActiveInverted,u[`--n-item-text-color-active-hover`]=i.itemTextColorActiveHoverInverted,u[`--n-item-icon-color`]=i.itemIconColorInverted,u[`--n-item-icon-color-hover`]=i.itemIconColorHoverInverted,u[`--n-item-icon-color-active`]=i.itemIconColorActiveInverted,u[`--n-item-icon-color-active-hover`]=i.itemIconColorActiveHoverInverted,u[`--n-item-icon-color-child-active`]=i.itemIconColorChildActiveInverted,u[`--n-item-icon-color-child-active-hover`]=i.itemIconColorChildActiveHoverInverted,u[`--n-item-icon-color-collapsed`]=i.itemIconColorCollapsedInverted,u[`--n-item-text-color-horizontal`]=i.itemTextColorHorizontalInverted,u[`--n-item-text-color-hover-horizontal`]=i.itemTextColorHoverHorizontalInverted,u[`--n-item-text-color-active-horizontal`]=i.itemTextColorActiveHorizontalInverted,u[`--n-item-text-color-child-active-horizontal`]=i.itemTextColorChildActiveHorizontalInverted,u[`--n-item-text-color-child-active-hover-horizontal`]=i.itemTextColorChildActiveHoverHorizontalInverted,u[`--n-item-text-color-active-hover-horizontal`]=i.itemTextColorActiveHoverHorizontalInverted,u[`--n-item-icon-color-horizontal`]=i.itemIconColorHorizontalInverted,u[`--n-item-icon-color-hover-horizontal`]=i.itemIconColorHoverHorizontalInverted,u[`--n-item-icon-color-active-horizontal`]=i.itemIconColorActiveHorizontalInverted,u[`--n-item-icon-color-active-hover-horizontal`]=i.itemIconColorActiveHoverHorizontalInverted,u[`--n-item-icon-color-child-active-horizontal`]=i.itemIconColorChildActiveHorizontalInverted,u[`--n-item-icon-color-child-active-hover-horizontal`]=i.itemIconColorChildActiveHoverHorizontalInverted,u[`--n-arrow-color`]=i.arrowColorInverted,u[`--n-arrow-color-hover`]=i.arrowColorHoverInverted,u[`--n-arrow-color-active`]=i.arrowColorActiveInverted,u[`--n-arrow-color-active-hover`]=i.arrowColorActiveHoverInverted,u[`--n-arrow-color-child-active`]=i.arrowColorChildActiveInverted,u[`--n-arrow-color-child-active-hover`]=i.arrowColorChildActiveHoverInverted,u[`--n-item-color-hover`]=i.itemColorHoverInverted,u[`--n-item-color-active`]=i.itemColorActiveInverted,u[`--n-item-color-active-hover`]=i.itemColorActiveHoverInverted,u[`--n-item-color-active-collapsed`]=i.itemColorActiveCollapsedInverted):(u[`--n-group-text-color`]=i.groupTextColor,u[`--n-color`]=i.color,u[`--n-item-text-color`]=i.itemTextColor,u[`--n-item-text-color-hover`]=i.itemTextColorHover,u[`--n-item-text-color-active`]=i.itemTextColorActive,u[`--n-item-text-color-child-active`]=i.itemTextColorChildActive,u[`--n-item-text-color-child-active-hover`]=i.itemTextColorChildActiveHover,u[`--n-item-text-color-active-hover`]=i.itemTextColorActiveHover,u[`--n-item-icon-color`]=i.itemIconColor,u[`--n-item-icon-color-hover`]=i.itemIconColorHover,u[`--n-item-icon-color-active`]=i.itemIconColorActive,u[`--n-item-icon-color-active-hover`]=i.itemIconColorActiveHover,u[`--n-item-icon-color-child-active`]=i.itemIconColorChildActive,u[`--n-item-icon-color-child-active-hover`]=i.itemIconColorChildActiveHover,u[`--n-item-icon-color-collapsed`]=i.itemIconColorCollapsed,u[`--n-item-text-color-horizontal`]=i.itemTextColorHorizontal,u[`--n-item-text-color-hover-horizontal`]=i.itemTextColorHoverHorizontal,u[`--n-item-text-color-active-horizontal`]=i.itemTextColorActiveHorizontal,u[`--n-item-text-color-child-active-horizontal`]=i.itemTextColorChildActiveHorizontal,u[`--n-item-text-color-child-active-hover-horizontal`]=i.itemTextColorChildActiveHoverHorizontal,u[`--n-item-text-color-active-hover-horizontal`]=i.itemTextColorActiveHoverHorizontal,u[`--n-item-icon-color-horizontal`]=i.itemIconColorHorizontal,u[`--n-item-icon-color-hover-horizontal`]=i.itemIconColorHoverHorizontal,u[`--n-item-icon-color-active-horizontal`]=i.itemIconColorActiveHorizontal,u[`--n-item-icon-color-active-hover-horizontal`]=i.itemIconColorActiveHoverHorizontal,u[`--n-item-icon-color-child-active-horizontal`]=i.itemIconColorChildActiveHorizontal,u[`--n-item-icon-color-child-active-hover-horizontal`]=i.itemIconColorChildActiveHoverHorizontal,u[`--n-arrow-color`]=i.arrowColor,u[`--n-arrow-color-hover`]=i.arrowColorHover,u[`--n-arrow-color-active`]=i.arrowColorActive,u[`--n-arrow-color-active-hover`]=i.arrowColorActiveHover,u[`--n-arrow-color-child-active`]=i.arrowColorChildActive,u[`--n-arrow-color-child-active-hover`]=i.arrowColorChildActiveHover,u[`--n-item-color-hover`]=i.itemColorHover,u[`--n-item-color-active`]=i.itemColorActive,u[`--n-item-color-active-hover`]=i.itemColorActiveHover,u[`--n-item-color-active-collapsed`]=i.itemColorActiveCollapsed),u}),T=n?M(`menu`,N(()=>e.inverted?`a`:`b`),w,e):void 0,E=me(),D=L(null),O=L(null),k=!0,A=()=>{var e;k?k=!1:(e=D.value)==null||e.sync({showAllItemsBeforeCalculate:!0})};function j(){return document.getElementById(E)}let P=L(-1);function ee(t){P.value=e.options.length-t}function I(e){e||(P.value=-1)}let te=N(()=>{let t=P.value;return{children:t===-1?[]:e.options.slice(t)}}),ne=N(()=>{let{childrenField:t,disabledField:n,keyField:r}=e;return pe([te.value],{getIgnored(e){return yt(e)},getChildren(e){return e[t]},getDisabled(e){return e[n]},getKey(e){return e[r]??e.name}})}),re=N(()=>pe([{}]).treeNodes[0]);function B(){if(P.value===-1)return o(wt,{root:!0,level:0,key:`__ellpisisGroupPlaceholder__`,internalKey:`__ellpisisGroupPlaceholder__`,title:`···`,tmNode:re.value,domId:E,isEllipsisPlaceholder:!0});let e=ne.value.treeNodes[0],t=y.value;return o(wt,{level:0,root:!0,key:`__ellpisisGroup__`,internalKey:`__ellpisisGroup__`,title:`···`,virtualChildActive:!!e.children?.some(e=>t.includes(e.key)),tmNode:e,domId:E,rawNodes:e.rawNode.children||[],tmNodes:e.children||[],isEllipsisPlaceholder:!0})}return{mergedClsPrefix:t,controlledExpandedKeys:g,uncontrolledExpanededKeys:p,mergedExpandedKeys:_,uncontrolledValue:d,mergedValue:f,activePath:y,tmNodes:v,mergedTheme:r,mergedCollapsed:a,cssVars:n?void 0:w,themeClass:T?.themeClass,overflowRef:D,counterRef:O,updateCounter:()=>{},onResize:A,onUpdateOverflow:I,onUpdateCount:ee,renderCounter:B,getCounter:j,onRender:T?.onRender,showOption:C,deriveResponsiveState:A}},render(){let{mergedClsPrefix:e,mode:t,themeClass:r,onRender:i}=this;i?.();let a=()=>this.tmNodes.map(e=>xt(e,this.$props)),s=t===`horizontal`&&this.responsive,c=()=>o(`div`,n(this.$attrs,{role:t===`horizontal`?`menubar`:`menu`,class:[`${e}-menu`,r,`${e}-menu--${t}`,s&&`${e}-menu--responsive`,this.mergedCollapsed&&`${e}-menu--collapsed`],style:this.cssVars}),s?o(fe,{ref:`overflowRef`,onUpdateOverflow:this.onUpdateOverflow,getCounter:this.getCounter,onUpdateCount:this.onUpdateCount,updateCounter:this.updateCounter,style:{width:`100%`,display:`flex`,overflow:`hidden`}},{default:a,counter:this.renderCounter}):a());return s?o(g,{onResize:this.onResize},{default:c}):c()}}),Et={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},Dt=V({name:`ArchiveOutline`,render:function(e,t){return y(),D(`svg`,Et,t[0]||=[E(`path`,{d:`M80 152v256a40.12 40.12 0 0 0 40 40h272a40.12 40.12 0 0 0 40-40V152`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),E(`rect`,{x:`48`,y:`64`,width:`416`,height:`80`,rx:`28`,ry:`28`,fill:`none`,stroke:`currentColor`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),E(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M320 304l-64 64l-64-64`},null,-1),E(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M256 345.89V224`},null,-1)])}}),Ot={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},kt=V({name:`CubeOutline`,render:function(e,t){return y(),D(`svg`,Ot,t[0]||=[E(`path`,{d:`M448 341.37V170.61A32 32 0 0 0 432.11 143l-152-88.46a47.94 47.94 0 0 0-48.24 0L79.89 143A32 32 0 0 0 64 170.61v170.76A32 32 0 0 0 79.89 369l152 88.46a48 48 0 0 0 48.24 0l152-88.46A32 32 0 0 0 448 341.37z`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),E(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M69 153.99l187 110l187-110`},null,-1),E(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M256 463.99v-200`},null,-1)])}}),At={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},jt=V({name:`DocumentTextOutline`,render:function(e,t){return y(),D(`svg`,At,t[0]||=[E(`path`,{d:`M416 221.25V416a48 48 0 0 1-48 48H144a48 48 0 0 1-48-48V96a48 48 0 0 1 48-48h98.75a32 32 0 0 1 22.62 9.37l141.26 141.26a32 32 0 0 1 9.37 22.62z`,fill:`none`,stroke:`currentColor`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),E(`path`,{d:`M256 56v120a32 32 0 0 0 32 32h120`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),E(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M176 288h160`},null,-1),E(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M176 368h160`},null,-1)])}}),Mt={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},Nt=V({name:`EarthOutline`,render:function(e,t){return y(),D(`svg`,Mt,t[0]||=[oe(`<path d="M464 256c0-114.87-93.13-208-208-208S48 141.13 48 256s93.13 208 208 208s208-93.13 208-208z" fill="none" stroke="currentColor" stroke-miterlimit="10" stroke-width="32"></path><path d="M445.57 172.14c-16.06.1-14.48 29.73-34.49 15.75c-7.43-5.18-12-12.71-21.33-15c-8.15-2-16.5.08-24.55 1.47c-9.15 1.58-20 2.29-26.94 9.22c-6.71 6.67-10.26 15.62-17.4 22.33c-13.81 13-19.64 27.19-10.7 45.57c8.6 17.67 26.59 27.26 46 26c19.07-1.27 38.88-12.33 38.33 15.38c-.2 9.8 1.85 16.6 4.86 25.71c2.79 8.4 2.6 16.54 3.24 25.21c1.18 16.2 4.16 34.36 12.2 48.67l15-21.16c1.85-2.62 5.72-6.29 6.64-9.38c1.63-5.47-1.58-14.87-1.95-21s-.19-12.34-1.13-18.47c-1.32-8.59-6.4-16.64-7.1-25.13c-1.29-15.81 1.6-28.43-10.58-41.65c-11.76-12.75-29-15.81-45.47-13.22c-8.3 1.3-41.71 6.64-28.3-12.33c2.65-3.73 7.28-6.79 10.26-10.34c2.59-3.09 4.84-8.77 7.88-11.18s17-5.18 21-3.95s8.17 7 11.64 9.56a49.89 49.89 0 0 0 21.81 9.36c13.66 2 42.22-5.94 42-23.46c-.04-8.4-7.84-20.1-10.92-27.96z" fill="currentColor"></path><path d="M287.45 316.3c-5.33-22.44-35.82-29.94-52.26-42.11c-9.45-7-17.86-17.81-30.27-18.69c-5.72-.41-10.51.83-16.18-.64c-5.2-1.34-9.28-4.14-14.82-3.41c-10.35 1.36-16.88 12.42-28 10.92c-10.55-1.42-21.42-13.76-23.82-23.81c-3.08-12.92 7.14-17.11 18.09-18.26c4.57-.48 9.7-1 14.09.67c5.78 2.15 8.51 7.81 13.7 10.67c9.73 5.33 11.7-3.19 10.21-11.83c-2.23-12.94-4.83-18.22 6.71-27.12c8-6.14 14.84-10.58 13.56-21.61c-.76-6.48-4.31-9.41-1-15.86c2.51-4.91 9.4-9.34 13.89-12.27c11.59-7.56 49.65-7 34.1-28.16c-4.57-6.21-13-17.31-21-18.83c-10-1.89-14.44 9.27-21.41 14.19c-7.2 5.09-21.22 10.87-28.43 3c-9.7-10.59 6.43-14.07 10-21.46s-8.27-21.36-14.61-24.9l-29.81 33.43a41.52 41.52 0 0 0 8.34 31.86c5.93 7.63 15.37 10.08 15.8 20.5c.42 10-1.14 15.12-7.68 22.15c-2.83 3-4.83 7.26-7.71 10.07c-3.53 3.43-2.22 2.38-7.73 3.32c-10.36 1.75-19.18 4.45-29.19 7.21C95.34 199.94 93.8 172.69 86.2 162l-25 20.19c-.27 3.31 4.1 9.4 5.29 13c6.83 20.57 20.61 36.48 29.51 56.16c9.37 20.84 34.53 15.06 45.64 33.32c9.86 16.2-.67 36.71 6.71 53.67c5.36 12.31 18 15 26.72 24c8.91 9.09 8.72 21.53 10.08 33.36a305.22 305.22 0 0 0 7.45 41.28c1.21 4.69 2.32 10.89 5.53 14.76c2.2 2.66 9.75 4.95 6.7 5.83c4.26.7 11.85 4.68 15.4 1.76c4.68-3.84 3.43-15.66 4.24-21c2.43-15.9 10.39-31.45 21.13-43.35c10.61-11.74 25.15-19.69 34.11-33c8.73-12.98 11.36-30.49 7.74-45.68zm-33.39 26.32c-6 10.71-19.36 17.88-27.95 26.39c-2.33 2.31-7.29 10.31-10.21 8.58c-2.09-1.24-2.8-11.62-3.57-14a61.17 61.17 0 0 0-21.71-29.95c-3.13-2.37-10.89-5.45-12.68-8.7c-2-3.53-.2-11.86-.13-15.7c.11-5.6-2.44-14.91-1.06-20c1.6-5.87-1.48-2.33 3.77-3.49c2.77-.62 14.21 1.39 17.66 2.11c5.48 1.14 8.5 4.55 12.82 8c11.36 9.11 23.87 16.16 36.6 23.14c9.86 5.46 12.76 12.37 6.46 23.62z" fill="currentColor"></path><path d="M184.46 67.09c4.74 4.63 9.2 10.11 16.27 10.57c6.69.45 13-3.17 18.84 1.38c6.48 5 11.15 11.33 19.75 12.89c8.32 1.51 17.13-3.35 19.19-11.86c2-8.11-2.31-16.93-2.57-25.07c0-1.13.61-6.15-.17-7c-.58-.64-5.42.08-6.16.1q-8.13.24-16.22 1.12a207.1 207.1 0 0 0-57.18 14.65c2.43 1.68 5.48 2.35 8.25 3.22z" fill="currentColor"></path><path d="M356.4 123.27c8.49 0 17.11-3.8 14.37-13.62c-2.3-8.23-6.22-17.16-15.76-12.72c-6.07 2.82-14.67 10-15.38 17.12c-.81 8.08 11.11 9.22 16.77 9.22z" fill="currentColor"></path><path d="M349.62 166.24c8.67 5.19 21.53 2.75 28.07-4.66c5.11-5.8 8.12-15.87 17.31-15.86a15.4 15.4 0 0 1 10.82 4.41c3.8 3.93 3.05 7.62 3.86 12.54c1.81 11.05 13.66.63 16.75-3.65c2-2.79 4.71-6.93 3.8-10.56c-.84-3.39-4.8-7-6.56-10.11c-5.14-9-9.37-19.47-17.07-26.74c-7.41-7-16.52-6.19-23.55 1.08c-5.76 6-12.45 10.75-16.39 18.05c-2.78 5.13-5.91 7.58-11.54 8.91c-3.1.73-6.64 1-9.24 3.08c-7.24 5.7-3.12 19.39 3.74 23.51z" fill="currentColor"></path>`,6)])}}),Pt={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},Ft=V({name:`GitNetworkOutline`,render:function(e,t){return y(),D(`svg`,Pt,t[0]||=[oe(`<circle cx="128" cy="96" r="48" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"></circle><circle cx="256" cy="416" r="48" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"></circle><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32" d="M256 256v112"></path><circle cx="384" cy="96" r="48" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"></circle><path d="M128 144c0 74.67 68.92 112 128 112" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"></path><path d="M384 144c0 74.67-68.92 112-128 112" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"></path>`,6)])}}),It={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},Lt=V({name:`GridOutline`,render:function(e,t){return y(),D(`svg`,It,t[0]||=[E(`rect`,{x:`48`,y:`48`,width:`176`,height:`176`,rx:`20`,ry:`20`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),E(`rect`,{x:`288`,y:`48`,width:`176`,height:`176`,rx:`20`,ry:`20`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),E(`rect`,{x:`48`,y:`288`,width:`176`,height:`176`,rx:`20`,ry:`20`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),E(`rect`,{x:`288`,y:`288`,width:`176`,height:`176`,rx:`20`,ry:`20`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1)])}}),Rt={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},zt=V({name:`LogOutOutline`,render:function(e,t){return y(),D(`svg`,Rt,t[0]||=[E(`path`,{d:`M304 336v40a40 40 0 0 1-40 40H104a40 40 0 0 1-40-40V136a40 40 0 0 1 40-40h152c22.09 0 48 17.91 48 40v40`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),E(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M368 336l80-80l-80-80`},null,-1),E(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M176 256h256`},null,-1)])}}),Bt={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},Vt=V({name:`ReaderOutline`,render:function(e,t){return y(),D(`svg`,Bt,t[0]||=[E(`rect`,{x:`96`,y:`48`,width:`320`,height:`416`,rx:`48`,ry:`48`,fill:`none`,stroke:`currentColor`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1),E(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M176 128h160`},null,-1),E(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M176 208h160`},null,-1),E(`path`,{fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`,d:`M176 288h80`},null,-1)])}}),Ht={xmlns:`http://www.w3.org/2000/svg`,"xmlns:xlink":`http://www.w3.org/1999/xlink`,viewBox:`0 0 512 512`},Ut=V({name:`SettingsOutline`,render:function(e,t){return y(),D(`svg`,Ht,t[0]||=[E(`path`,{d:`M262.29 192.31a64 64 0 1 0 57.4 57.4a64.13 64.13 0 0 0-57.4-57.4zM416.39 256a154.34 154.34 0 0 1-1.53 20.79l45.21 35.46a10.81 10.81 0 0 1 2.45 13.75l-42.77 74a10.81 10.81 0 0 1-13.14 4.59l-44.9-18.08a16.11 16.11 0 0 0-15.17 1.75A164.48 164.48 0 0 1 325 400.8a15.94 15.94 0 0 0-8.82 12.14l-6.73 47.89a11.08 11.08 0 0 1-10.68 9.17h-85.54a11.11 11.11 0 0 1-10.69-8.87l-6.72-47.82a16.07 16.07 0 0 0-9-12.22a155.3 155.3 0 0 1-21.46-12.57a16 16 0 0 0-15.11-1.71l-44.89 18.07a10.81 10.81 0 0 1-13.14-4.58l-42.77-74a10.8 10.8 0 0 1 2.45-13.75l38.21-30a16.05 16.05 0 0 0 6-14.08c-.36-4.17-.58-8.33-.58-12.5s.21-8.27.58-12.35a16 16 0 0 0-6.07-13.94l-38.19-30A10.81 10.81 0 0 1 49.48 186l42.77-74a10.81 10.81 0 0 1 13.14-4.59l44.9 18.08a16.11 16.11 0 0 0 15.17-1.75A164.48 164.48 0 0 1 187 111.2a15.94 15.94 0 0 0 8.82-12.14l6.73-47.89A11.08 11.08 0 0 1 213.23 42h85.54a11.11 11.11 0 0 1 10.69 8.87l6.72 47.82a16.07 16.07 0 0 0 9 12.22a155.3 155.3 0 0 1 21.46 12.57a16 16 0 0 0 15.11 1.71l44.89-18.07a10.81 10.81 0 0 1 13.14 4.58l42.77 74a10.8 10.8 0 0 1-2.45 13.75l-38.21 30a16.05 16.05 0 0 0-6.05 14.08c.33 4.14.55 8.3.55 12.47z`,fill:`none`,stroke:`currentColor`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"stroke-width":`32`},null,-1)])}}),Wt={key:0,class:`brand-copy`},Gt={class:`account`},Kt={class:`account-copy`},qt={class:`update-banner-body`},Jt=V({__name:`AppLayout`,setup(e){let t=ke(),n=Ee(),i=Ae(),a=_e(),s=L(window.innerWidth<900);function c(e,t,n){return{label:()=>o(Oe,{to:{name:t}},{default:()=>e}),key:t,icon:()=>o(Ce,null,{default:()=>o(n)})}}let d=[c(`运行概览`,`dashboard`,Lt),c(`代理编排`,`orchestration`,Ft),c(`配置管理`,`config`,jt),c(`配置能力`,`schema`,Me),c(`dae 版本`,`versions`,kt),c(`Geo 数据`,`geo`,Nt),c(`运行日志`,`logs`,Vt),c(`配置备份`,`backups`,Dt),c(`面板设置`,`settings`,Ut)],p=N(()=>String(t.name||`dashboard`)),m=N(()=>String(t.meta.title||`kdae-panel`));async function h(){try{await i.logout(),await n.replace({name:`login`})}catch(e){a.error(e instanceof Error?e.message:`退出登录失败`)}}function g(){i.clearSession(),n.replace({name:`login`}),a.warning(`登录会话已过期，请重新登录`)}function v(){window.innerWidth<900&&(s.value=!0)}let x=L(null),S=L(!1);async function C(){try{x.value=await P(`/api/v1/panel/update`)}catch{x.value=null}}function w(e){let t=e.detail;x.value&&t&&(x.value.status=t)}return u(()=>{window.addEventListener(`kdae-panel:auth-expired`,g),window.addEventListener(`kdae-panel:self-update-changed`,w),window.addEventListener(`resize`,v),C()}),_(()=>{window.removeEventListener(`kdae-panel:auth-expired`,g),window.removeEventListener(`kdae-panel:self-update-changed`,w),window.removeEventListener(`resize`,v)}),(e,t)=>{let n=l(`RouterView`);return y(),ie(O(qe),{"has-sider":``,class:`app-shell`},{default:f(()=>[r(O(nt),{bordered:``,"collapse-mode":`width`,"collapsed-width":64,width:236,collapsed:s.value,"show-trigger":`bar`,onCollapse:t[0]||=e=>s.value=!0,onExpand:t[1]||=e=>s.value=!1},{default:f(()=>[E(`div`,{class:te([`brand`,{compact:s.value}])},[t[4]||=E(`div`,{class:`brand-mark`},`K`,-1),s.value?le(``,!0):(y(),D(`div`,Wt,[...t[3]||=[E(`strong`,null,`kdae-panel`,-1),E(`span`,null,`零侵入管理面板`,-1)]]))],2),r(O(Tt),{value:p.value,collapsed:s.value,"collapsed-width":64,"collapsed-icon-size":22,options:d},null,8,[`value`,`collapsed`])]),_:1},8,[`collapsed`]),r(O(qe),null,{default:f(()=>[r(O(Ze),{bordered:``,class:`app-header`},{default:f(()=>[E(`div`,null,[r(O(ve),{depth:`3`,class:`eyebrow`},{default:f(()=>[...t[5]||=[b(`KDAE CONTROL PLANE`,-1)]]),_:1}),E(`h1`,null,B(m.value),1)]),E(`div`,Gt,[r(O(Be),{round:``,size:`small`},{default:f(()=>[b(B(O(i).user?.username?.slice(0,1).toUpperCase()),1)]),_:1}),E(`div`,Kt,[E(`strong`,null,B(O(i).user?.username),1),t[6]||=E(`span`,null,`管理员`,-1)]),r(O(A),{quaternary:``,circle:``,title:`退出登录`,onClick:h},{icon:f(()=>[r(O(Ce),null,{default:f(()=>[r(O(zt))]),_:1})]),_:1})])]),_:1}),r(O(Je),{class:`app-content`,"content-style":`padding: 28px;`},{default:f(()=>[x.value?.check.updateAvailable&&!S.value?(y(),ie(O(we),{key:0,type:`info`,closable:``,class:`update-banner`,onClose:t[2]||=e=>S.value=!0},{default:f(()=>[E(`div`,qt,[E(`span`,null,[t[7]||=b(` 面板有新版本 `,-1),E(`strong`,null,B(x.value.check.latest),1),b(`（当前 `+B(x.value.check.current)+`）。 `,1),x.value.status?.enabled&&x.value.status.updatable?(y(),D(j,{key:0},[b(`升级会替换面板二进制并重启自身，配置与账号数据都会保留。`)],64)):x.value.status&&!x.value.status.enabled?(y(),D(j,{key:1},[b(`可直接在这里启用一键升级，不需要 SSH。`)],64)):x.value.status?.problem?(y(),D(j,{key:2},[b(`当前无法一键升级：`+B(x.value.status.problem),1)],64)):(y(),D(j,{key:3},[b(`当前部署不支持一键升级，可重新执行一键部署命令。`)],64)),t[8]||=E(`a`,{href:`https://github.com/tuoro/kdae-panel/releases/latest`,target:`_blank`,rel:`noopener`},`查看发布说明`,-1)]),r(Ne,{payload:x.value,label:`立即升级`},null,8,[`payload`])])]),_:1})):le(``,!0),r(n)]),_:1})]),_:1})]),_:1})}}});export{Jt as default};