import{$t as e,Dt as t,En as n,Jt as r,Nt as i,P as a,Qt as o,Rt as s,Wt as c,Yt as l,Zt as u,en as d,ft as f,gn as p,j as m,nr as h,or as g,pt as _,wn as v,wt as y,x as b,xt as x,zn as S}from"./client-DzOxLNa2.js";import{t as C}from"./Close-BbKQzVO9.js";import{E as w}from"./index-DZhlw9Fw.js";function T(e){let{textColor2:t,primaryColorHover:n,primaryColorPressed:r,primaryColor:i,infoColor:a,successColor:o,warningColor:c,errorColor:l,baseColor:u,borderColor:d,opacityDisabled:f,tagColor:p,closeIconColor:m,closeIconColorHover:h,closeIconColorPressed:g,borderRadiusSmall:_,fontSizeMini:v,fontSizeTiny:y,fontSizeSmall:b,fontSizeMedium:x,heightMini:S,heightTiny:C,heightSmall:T,heightMedium:E,closeColorHover:D,closeColorPressed:O,buttonColor2Hover:k,buttonColor2Pressed:A,fontWeightStrong:j}=e;return Object.assign(Object.assign({},w),{closeBorderRadius:_,heightTiny:S,heightSmall:C,heightMedium:T,heightLarge:E,borderRadius:_,opacityDisabled:f,fontSizeTiny:v,fontSizeSmall:y,fontSizeMedium:b,fontSizeLarge:x,fontWeightStrong:j,textColorCheckable:t,textColorHoverCheckable:t,textColorPressedCheckable:t,textColorChecked:u,colorCheckable:`#0000`,colorHoverCheckable:k,colorPressedCheckable:A,colorChecked:i,colorCheckedHover:n,colorCheckedPressed:r,border:`1px solid ${d}`,textColor:t,color:p,colorBordered:`rgb(250, 250, 252)`,closeIconColor:m,closeIconColorHover:h,closeIconColorPressed:g,closeColorHover:D,closeColorPressed:O,borderPrimary:`1px solid ${s(i,{alpha:.3})}`,textColorPrimary:i,colorPrimary:s(i,{alpha:.12}),colorBorderedPrimary:s(i,{alpha:.1}),closeIconColorPrimary:i,closeIconColorHoverPrimary:i,closeIconColorPressedPrimary:i,closeColorHoverPrimary:s(i,{alpha:.12}),closeColorPressedPrimary:s(i,{alpha:.18}),borderInfo:`1px solid ${s(a,{alpha:.3})}`,textColorInfo:a,colorInfo:s(a,{alpha:.12}),colorBorderedInfo:s(a,{alpha:.1}),closeIconColorInfo:a,closeIconColorHoverInfo:a,closeIconColorPressedInfo:a,closeColorHoverInfo:s(a,{alpha:.12}),closeColorPressedInfo:s(a,{alpha:.18}),borderSuccess:`1px solid ${s(o,{alpha:.3})}`,textColorSuccess:o,colorSuccess:s(o,{alpha:.12}),colorBorderedSuccess:s(o,{alpha:.1}),closeIconColorSuccess:o,closeIconColorHoverSuccess:o,closeIconColorPressedSuccess:o,closeColorHoverSuccess:s(o,{alpha:.12}),closeColorPressedSuccess:s(o,{alpha:.18}),borderWarning:`1px solid ${s(c,{alpha:.35})}`,textColorWarning:c,colorWarning:s(c,{alpha:.15}),colorBorderedWarning:s(c,{alpha:.12}),closeIconColorWarning:c,closeIconColorHoverWarning:c,closeIconColorPressedWarning:c,closeColorHoverWarning:s(c,{alpha:.12}),closeColorPressedWarning:s(c,{alpha:.18}),borderError:`1px solid ${s(l,{alpha:.23})}`,textColorError:l,colorError:s(l,{alpha:.1}),colorBorderedError:s(l,{alpha:.08}),closeIconColorError:l,closeIconColorHoverError:l,closeIconColorPressedError:l,closeColorHoverError:s(l,{alpha:.12}),closeColorPressedError:s(l,{alpha:.18})})}var E={name:`Tag`,common:b,self:T},D={color:Object,type:{type:String,default:`default`},round:Boolean,size:String,closable:Boolean,disabled:{type:Boolean,default:void 0}},O=l(`tag`,`
 --n-close-margin: var(--n-close-margin-top) var(--n-close-margin-right) var(--n-close-margin-bottom) var(--n-close-margin-left);
 white-space: nowrap;
 position: relative;
 box-sizing: border-box;
 cursor: default;
 display: inline-flex;
 align-items: center;
 flex-wrap: nowrap;
 padding: var(--n-padding);
 border-radius: var(--n-border-radius);
 color: var(--n-text-color);
 background-color: var(--n-color);
 transition: 
 border-color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier),
 opacity .3s var(--n-bezier);
 line-height: 1;
 height: var(--n-height);
 font-size: var(--n-font-size);
`,[o(`strong`,`
 font-weight: var(--n-font-weight-strong);
 `),u(`border`,`
 pointer-events: none;
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 border-radius: inherit;
 border: var(--n-border);
 transition: border-color .3s var(--n-bezier);
 `),u(`icon`,`
 display: flex;
 margin: 0 4px 0 0;
 color: var(--n-text-color);
 transition: color .3s var(--n-bezier);
 font-size: var(--n-avatar-size-override);
 `),u(`avatar`,`
 display: flex;
 margin: 0 6px 0 0;
 `),u(`close`,`
 margin: var(--n-close-margin);
 transition:
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 `),o(`round`,`
 padding: 0 calc(var(--n-height) / 3);
 border-radius: calc(var(--n-height) / 2);
 `,[u(`icon`,`
 margin: 0 4px 0 calc((var(--n-height) - 8px) / -2);
 `),u(`avatar`,`
 margin: 0 6px 0 calc((var(--n-height) - 8px) / -2);
 `),o(`closable`,`
 padding: 0 calc(var(--n-height) / 4) 0 calc(var(--n-height) / 3);
 `)]),o(`icon, avatar`,[o(`round`,`
 padding: 0 calc(var(--n-height) / 3) 0 calc(var(--n-height) / 2);
 `)]),o(`disabled`,`
 cursor: not-allowed !important;
 opacity: var(--n-opacity-disabled);
 `),o(`checkable`,`
 cursor: pointer;
 box-shadow: none;
 color: var(--n-text-color-checkable);
 background-color: var(--n-color-checkable);
 `,[e(`disabled`,[r(`&:hover`,`background-color: var(--n-color-hover-checkable);`,[e(`checked`,`color: var(--n-text-color-hover-checkable);`)]),r(`&:active`,`background-color: var(--n-color-pressed-checkable);`,[e(`checked`,`color: var(--n-text-color-pressed-checkable);`)])]),o(`checked`,`
 color: var(--n-text-color-checked);
 background-color: var(--n-color-checked);
 `,[e(`disabled`,[r(`&:hover`,`background-color: var(--n-color-checked-hover);`),r(`&:active`,`background-color: var(--n-color-checked-pressed);`)])])])]),k=Object.assign(Object.assign(Object.assign({},m.props),D),{bordered:{type:Boolean,default:void 0},checked:Boolean,checkable:Boolean,strong:Boolean,triggerClickOnClose:Boolean,onClose:[Array,Function],onMouseenter:Function,onMouseleave:Function,"onUpdate:checked":Function,onUpdateChecked:Function,internalCloseFocusable:{type:Boolean,default:!0},internalCloseIsButtonTag:{type:Boolean,default:!0},onCheckedChange:Function}),A=i(`n-tag`),j=v({name:`Tag`,props:k,slots:Object,setup(e){let n=h(null),{mergedBorderedRef:r,mergedClsPrefixRef:i,inlineThemeDisabled:o,mergedRtlRef:s,mergedComponentPropsRef:l}=_(e),u=p(()=>e.size||l?.value?.Tag?.size||`medium`),v=m(`Tag`,`-tag`,O,E,e,i);S(A,{roundRef:g(e,`round`)});function b(){if(!e.disabled&&e.checkable){let{checked:t,onCheckedChange:n,onUpdateChecked:r,"onUpdate:checked":i}=e;r&&r(!t),i&&i(!t),n&&n(!t)}}function x(t){if(e.triggerClickOnClose||t.stopPropagation(),!e.disabled){let{onClose:n}=e;n&&y(n,t)}}let C={setTextContent(e){let{value:t}=n;t&&(t.textContent=e)}},w=a(`Tag`,s,i),T=p(()=>{let{type:t,color:{color:n,textColor:i}={}}=e,a=u.value,{common:{cubicBezierEaseInOut:o},self:{padding:s,closeMargin:l,borderRadius:f,opacityDisabled:p,textColorCheckable:m,textColorHoverCheckable:h,textColorPressedCheckable:g,textColorChecked:_,colorCheckable:y,colorHoverCheckable:b,colorPressedCheckable:x,colorChecked:S,colorCheckedHover:C,colorCheckedPressed:w,closeBorderRadius:T,fontWeightStrong:E,[d(`colorBordered`,t)]:D,[d(`closeSize`,a)]:O,[d(`closeIconSize`,a)]:k,[d(`fontSize`,a)]:A,[d(`height`,a)]:j,[d(`color`,t)]:M,[d(`textColor`,t)]:N,[d(`border`,t)]:P,[d(`closeIconColor`,t)]:F,[d(`closeIconColorHover`,t)]:I,[d(`closeIconColorPressed`,t)]:L,[d(`closeColorHover`,t)]:R,[d(`closeColorPressed`,t)]:z}}=v.value,B=c(l);return{"--n-font-weight-strong":E,"--n-avatar-size-override":`calc(${j} - 8px)`,"--n-bezier":o,"--n-border-radius":f,"--n-border":P,"--n-close-icon-size":k,"--n-close-color-pressed":z,"--n-close-color-hover":R,"--n-close-border-radius":T,"--n-close-icon-color":F,"--n-close-icon-color-hover":I,"--n-close-icon-color-pressed":L,"--n-close-icon-color-disabled":F,"--n-close-margin-top":B.top,"--n-close-margin-right":B.right,"--n-close-margin-bottom":B.bottom,"--n-close-margin-left":B.left,"--n-close-size":O,"--n-color":n||(r.value?D:M),"--n-color-checkable":y,"--n-color-checked":S,"--n-color-checked-hover":C,"--n-color-checked-pressed":w,"--n-color-hover-checkable":b,"--n-color-pressed-checkable":x,"--n-font-size":A,"--n-height":j,"--n-opacity-disabled":p,"--n-padding":s,"--n-text-color":i||N,"--n-text-color-checkable":m,"--n-text-color-checked":_,"--n-text-color-hover-checkable":h,"--n-text-color-pressed-checkable":g}}),D=o?f(`tag`,p(()=>{let n=``,{type:i,color:{color:a,textColor:o}={}}=e;return n+=i[0],n+=u.value[0],a&&(n+=`a${t(a)}`),o&&(n+=`b${t(o)}`),r.value&&(n+=`c`),n}),T,e):void 0;return Object.assign(Object.assign({},C),{rtlEnabled:w,mergedClsPrefix:i,contentRef:n,mergedBordered:r,handleClick:b,handleCloseClick:x,cssVars:o?void 0:T,themeClass:D?.themeClass,onRender:D?.onRender})},render(){var e;let{mergedClsPrefix:t,rtlEnabled:r,closable:i,color:{borderColor:a}={},round:o,onRender:s,$slots:c}=this;s?.();let l=x(c.avatar,e=>e&&n(`div`,{class:`${t}-tag__avatar`},e)),u=x(c.icon,e=>e&&n(`div`,{class:`${t}-tag__icon`},e));return n(`div`,{class:[`${t}-tag`,this.themeClass,{[`${t}-tag--rtl`]:r,[`${t}-tag--strong`]:this.strong,[`${t}-tag--disabled`]:this.disabled,[`${t}-tag--checkable`]:this.checkable,[`${t}-tag--checked`]:this.checkable&&this.checked,[`${t}-tag--round`]:o,[`${t}-tag--avatar`]:l,[`${t}-tag--icon`]:u,[`${t}-tag--closable`]:i}],style:this.cssVars,onClick:this.handleClick,onMouseenter:this.onMouseenter,onMouseleave:this.onMouseleave},u||l,n(`span`,{class:`${t}-tag__content`,ref:`contentRef`},(e=this.$slots).default?.call(e)),!this.checkable&&i?n(C,{clsPrefix:t,class:`${t}-tag__close`,disabled:this.disabled,onClick:this.handleCloseClick,focusable:this.internalCloseFocusable,round:o,isButtonTag:this.internalCloseIsButtonTag,absolute:!0}):null,!this.checkable&&this.mergedBordered?n(`div`,{class:`${t}-tag__border`,style:{borderColor:a}}):null)}});export{A as n,j as t};