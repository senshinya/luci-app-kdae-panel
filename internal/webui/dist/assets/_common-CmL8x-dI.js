import{Ct as e,En as t,Jt as n,P as r,Qt as i,Wt as a,Yt as o,Zt as s,_ as c,_t as l,en as u,ft as d,gn as f,j as p,nn as m,pt as h,qt as g,tn as _,wn as v,wt as y,x as b,xt as x}from"./client-DzOxLNa2.js";import{t as S}from"./Close-D2ABeYYO.js";var C={paddingSmall:`12px 16px 12px`,paddingMedium:`19px 24px 20px`,paddingLarge:`23px 32px 24px`,paddingHuge:`27px 40px 28px`,titleFontSizeSmall:`16px`,titleFontSizeMedium:`18px`,titleFontSizeLarge:`18px`,titleFontSizeHuge:`18px`,closeIconSize:`18px`,closeSize:`22px`};function w(e){let{primaryColor:t,borderRadius:n,lineHeight:r,fontSize:i,cardColor:a,textColor2:o,textColor1:s,dividerColor:c,fontWeightStrong:l,closeIconColor:u,closeIconColorHover:d,closeIconColorPressed:f,closeColorHover:p,closeColorPressed:m,modalColor:h,boxShadow1:g,popoverColor:_,actionColor:v}=e;return Object.assign(Object.assign({},C),{lineHeight:r,color:a,colorModal:h,colorPopover:_,colorTarget:t,colorEmbedded:v,colorEmbeddedModal:v,colorEmbeddedPopover:v,textColor:o,titleTextColor:s,borderColor:c,actionColor:v,titleFontWeight:l,closeColorHover:p,closeColorPressed:m,closeBorderRadius:n,closeIconColor:u,closeIconColorHover:d,closeIconColorPressed:f,fontSizeSmall:i,fontSizeMedium:i,fontSizeLarge:i,fontSizeHuge:i,boxShadow:g,borderRadius:n})}var T={name:`Card`,common:b,self:w},E=o(`card-content`,`
 flex: 1;
 min-width: 0;
 box-sizing: border-box;
 padding: 0 var(--n-padding-left) var(--n-padding-bottom) var(--n-padding-left);
 font-size: var(--n-font-size);
`),D=n([o(`card`,`
 font-size: var(--n-font-size);
 line-height: var(--n-line-height);
 display: flex;
 flex-direction: column;
 width: 100%;
 box-sizing: border-box;
 position: relative;
 border-radius: var(--n-border-radius);
 background-color: var(--n-color);
 color: var(--n-text-color);
 word-break: break-word;
 transition: 
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 `,[g({background:`var(--n-color-modal)`}),i(`hoverable`,[n(`&:hover`,`box-shadow: var(--n-box-shadow);`)]),i(`content-segmented`,[n(`>`,[o(`card-content`,`
 padding-top: var(--n-padding-bottom);
 `),s(`content-scrollbar`,[n(`>`,[o(`scrollbar-container`,[n(`>`,[o(`card-content`,`
 padding-top: var(--n-padding-bottom);
 `)])])])])])]),i(`content-soft-segmented`,[n(`>`,[o(`card-content`,`
 margin: 0 var(--n-padding-left);
 padding: var(--n-padding-bottom) 0;
 `),s(`content-scrollbar`,[n(`>`,[o(`scrollbar-container`,[n(`>`,[o(`card-content`,`
 margin: 0 var(--n-padding-left);
 padding: var(--n-padding-bottom) 0;
 `)])])])])])]),i(`footer-segmented`,[n(`>`,[s(`footer`,`
 padding-top: var(--n-padding-bottom);
 `)])]),i(`footer-soft-segmented`,[n(`>`,[s(`footer`,`
 padding: var(--n-padding-bottom) 0;
 margin: 0 var(--n-padding-left);
 `)])]),n(`>`,[o(`card-header`,`
 box-sizing: border-box;
 display: flex;
 align-items: center;
 font-size: var(--n-title-font-size);
 padding:
 var(--n-padding-top)
 var(--n-padding-left)
 var(--n-padding-bottom)
 var(--n-padding-left);
 `,[s(`main`,`
 font-weight: var(--n-title-font-weight);
 transition: color .3s var(--n-bezier);
 flex: 1;
 min-width: 0;
 color: var(--n-title-text-color);
 `),s(`extra`,`
 display: flex;
 align-items: center;
 font-size: var(--n-font-size);
 font-weight: 400;
 transition: color .3s var(--n-bezier);
 color: var(--n-text-color);
 `),s(`close`,`
 margin: 0 0 0 8px;
 transition:
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 `)]),s(`action`,`
 box-sizing: border-box;
 transition:
 background-color .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 background-clip: padding-box;
 background-color: var(--n-action-color);
 `),E,o(`card-content`,[n(`&:first-child`,`
 padding-top: var(--n-padding-bottom);
 `)]),s(`content-scrollbar`,`
 display: flex;
 flex-direction: column;
 `,[n(`>`,[o(`scrollbar-container`,[n(`>`,[E])])]),n(`&:first-child >`,[o(`scrollbar-container`,[n(`>`,[o(`card-content`,`
 padding-top: var(--n-padding-bottom);
 `)])])])]),s(`footer`,`
 box-sizing: border-box;
 padding: 0 var(--n-padding-left) var(--n-padding-bottom) var(--n-padding-left);
 font-size: var(--n-font-size);
 `,[n(`&:first-child`,`
 padding-top: var(--n-padding-bottom);
 `)]),s(`action`,`
 background-color: var(--n-action-color);
 padding: var(--n-padding-bottom) var(--n-padding-left);
 border-bottom-left-radius: var(--n-border-radius);
 border-bottom-right-radius: var(--n-border-radius);
 `)]),o(`card-cover`,`
 overflow: hidden;
 width: 100%;
 border-radius: var(--n-border-radius) var(--n-border-radius) 0 0;
 `,[n(`img`,`
 display: block;
 width: 100%;
 `)]),i(`bordered`,`
 border: 1px solid var(--n-border-color);
 `,[n(`&:target`,`border-color: var(--n-color-target);`)]),i(`action-segmented`,[n(`>`,[s(`action`,[n(`&:not(:first-child)`,`
 border-top: 1px solid var(--n-border-color);
 `)])])]),i(`content-segmented, content-soft-segmented`,[n(`>`,[o(`card-content`,`
 transition: border-color 0.3s var(--n-bezier);
 `,[n(`&:not(:first-child)`,`
 border-top: 1px solid var(--n-border-color);
 `)]),s(`content-scrollbar`,`
 transition: border-color 0.3s var(--n-bezier);
 `,[n(`&:not(:first-child)`,`
 border-top: 1px solid var(--n-border-color);
 `)])])]),i(`footer-segmented, footer-soft-segmented`,[n(`>`,[s(`footer`,`
 transition: border-color 0.3s var(--n-bezier);
 `,[n(`&:not(:first-child)`,`
 border-top: 1px solid var(--n-border-color);
 `)])])]),i(`embedded`,`
 background-color: var(--n-color-embedded);
 `)]),_(o(`card`,`
 background: var(--n-color-modal);
 `,[i(`embedded`,`
 background-color: var(--n-color-embedded-modal);
 `)])),m(o(`card`,`
 background: var(--n-color-popover);
 `,[i(`embedded`,`
 background-color: var(--n-color-embedded-popover);
 `)]))]),O={title:[String,Function],contentClass:String,contentStyle:[Object,String],contentScrollable:Boolean,headerClass:String,headerStyle:[Object,String],headerExtraClass:String,headerExtraStyle:[Object,String],footerClass:String,footerStyle:[Object,String],embedded:Boolean,segmented:{type:[Boolean,Object],default:!1},size:String,bordered:{type:Boolean,default:!0},closable:Boolean,hoverable:Boolean,role:String,onClose:[Function,Array],tag:{type:String,default:`div`},cover:Function,content:[String,Function],footer:Function,action:Function,headerExtra:Function,closeFocusable:Boolean},k=e(O),A=v({name:`Card`,props:Object.assign(Object.assign({},p.props),O),slots:Object,setup(e){let t=()=>{let{onClose:t}=e;t&&y(t)},{inlineThemeDisabled:n,mergedClsPrefixRef:i,mergedRtlRef:o,mergedComponentPropsRef:s}=h(e),c=p(`Card`,`-card`,D,T,e,i),l=r(`Card`,o,i),m=f(()=>e.size||s?.value?.Card?.size||`medium`),g=f(()=>{let e=m.value,{self:{color:t,colorModal:n,colorTarget:r,textColor:i,titleTextColor:o,titleFontWeight:s,borderColor:l,actionColor:d,borderRadius:f,lineHeight:p,closeIconColor:h,closeIconColorHover:g,closeIconColorPressed:_,closeColorHover:v,closeColorPressed:y,closeBorderRadius:b,closeIconSize:x,closeSize:S,boxShadow:C,colorPopover:w,colorEmbedded:T,colorEmbeddedModal:E,colorEmbeddedPopover:D,[u(`padding`,e)]:O,[u(`fontSize`,e)]:k,[u(`titleFontSize`,e)]:A},common:{cubicBezierEaseInOut:j}}=c.value,{top:M,left:N,bottom:P}=a(O);return{"--n-bezier":j,"--n-border-radius":f,"--n-color":t,"--n-color-modal":n,"--n-color-popover":w,"--n-color-embedded":T,"--n-color-embedded-modal":E,"--n-color-embedded-popover":D,"--n-color-target":r,"--n-text-color":i,"--n-line-height":p,"--n-action-color":d,"--n-title-text-color":o,"--n-title-font-weight":s,"--n-close-icon-color":h,"--n-close-icon-color-hover":g,"--n-close-icon-color-pressed":_,"--n-close-color-hover":v,"--n-close-color-pressed":y,"--n-border-color":l,"--n-box-shadow":C,"--n-padding-top":M,"--n-padding-bottom":P,"--n-padding-left":N,"--n-font-size":k,"--n-title-font-size":A,"--n-close-size":S,"--n-close-icon-size":x,"--n-close-border-radius":b}}),_=n?d(`card`,f(()=>m.value[0]),g,e):void 0;return{rtlEnabled:l,mergedClsPrefix:i,mergedTheme:c,handleCloseClick:t,cssVars:n?void 0:g,themeClass:_?.themeClass,onRender:_?.onRender}},render(){let{segmented:e,bordered:n,hoverable:r,mergedClsPrefix:i,rtlEnabled:a,onRender:o,embedded:s,tag:u,$slots:d}=this;return o?.(),t(u,{class:[`${i}-card`,this.themeClass,s&&`${i}-card--embedded`,{[`${i}-card--rtl`]:a,[`${i}-card--content-scrollable`]:this.contentScrollable,[`${i}-card--content${typeof e!=`boolean`&&e.content===`soft`?`-soft`:``}-segmented`]:e===!0||e!==!1&&e.content,[`${i}-card--footer${typeof e!=`boolean`&&e.footer===`soft`?`-soft`:``}-segmented`]:e===!0||e!==!1&&e.footer,[`${i}-card--action-segmented`]:e===!0||e!==!1&&e.action,[`${i}-card--bordered`]:n,[`${i}-card--hoverable`]:r}],style:this.cssVars,role:this.role},x(d.cover,e=>{let n=this.cover?l([this.cover()]):e;return n&&t(`div`,{class:`${i}-card-cover`,role:`none`},n)}),x(d.header,e=>{let{title:n}=this,r=n?l(typeof n==`function`?[n()]:[n]):e;return r||this.closable?t(`div`,{class:[`${i}-card-header`,this.headerClass],style:this.headerStyle,role:`heading`},t(`div`,{class:`${i}-card-header__main`,role:`heading`},r),x(d[`header-extra`],e=>{let n=this.headerExtra?l([this.headerExtra()]):e;return n&&t(`div`,{class:[`${i}-card-header__extra`,this.headerExtraClass],style:this.headerExtraStyle},n)}),this.closable&&t(S,{clsPrefix:i,class:`${i}-card-header__close`,onClick:this.handleCloseClick,focusable:this.closeFocusable,absolute:!0})):null}),x(d.default,e=>{let{content:n}=this,r=n?l(typeof n==`function`?[n()]:[n]):e;return r?this.contentScrollable?t(c,{class:`${i}-card__content-scrollbar`,contentClass:[`${i}-card-content`,this.contentClass],contentStyle:this.contentStyle},r):t(`div`,{class:[`${i}-card-content`,this.contentClass],style:this.contentStyle,role:`none`},r):null}),x(d.footer,e=>{let n=this.footer?l([this.footer()]):e;return n&&t(`div`,{class:[`${i}-card__footer`,this.footerClass],style:this.footerStyle,role:`none`},n)}),x(d.action,e=>{let n=this.action?l([this.action()]):e;return n&&t(`div`,{class:`${i}-card__action`,role:`none`},n)}))}}),j={gapSmall:`4px 8px`,gapMedium:`8px 12px`,gapLarge:`12px 16px`};export{T as a,O as i,A as n,w as o,k as r,j as t};