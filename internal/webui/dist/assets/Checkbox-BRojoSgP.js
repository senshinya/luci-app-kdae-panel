import{E as e,En as t,Ft as n,Jt as r,Lt as i,Nt as a,O as o,On as s,P as c,Qt as l,Yt as u,Zt as d,dt as f,en as p,ft as m,gn as h,j as g,nn as _,nr as v,or as y,pt as b,tn as x,wn as S,wt as C,xt as w,zn as T}from"./client-DzOxLNa2.js";import{t as E}from"./misc-DDs3MKLt.js";import{a as D}from"./text-DPkxR-eM.js";import{t as O}from"./light-DM3icw2V.js";var k=a(`n-checkbox-group`),A=S({name:`CheckboxGroup`,props:{min:Number,max:Number,size:String,value:Array,defaultValue:{type:Array,default:null},disabled:{type:Boolean,default:void 0},"onUpdate:value":[Function,Array],onUpdateValue:[Function,Array],onChange:[Function,Array]},setup(e){let{mergedClsPrefixRef:t}=b(e),n=f(e),{mergedSizeRef:r,mergedDisabledRef:i}=n,a=v(e.defaultValue),o=D(h(()=>e.value),a),s=h(()=>o.value?.length||0),c=h(()=>Array.isArray(o.value)?new Set(o.value):new Set);function l(t,r){let{nTriggerFormInput:i,nTriggerFormChange:s}=n,{onChange:c,"onUpdate:value":l,onUpdateValue:u}=e;if(Array.isArray(o.value)){let e=Array.from(o.value),n=e.findIndex(e=>e===r);t?~n||(e.push(r),u&&C(u,e,{actionType:`check`,value:r}),l&&C(l,e,{actionType:`check`,value:r}),i(),s(),a.value=e,c&&C(c,e)):~n&&(e.splice(n,1),u&&C(u,e,{actionType:`uncheck`,value:r}),l&&C(l,e,{actionType:`uncheck`,value:r}),c&&C(c,e),a.value=e,i(),s())}else t?(u&&C(u,[r],{actionType:`check`,value:r}),l&&C(l,[r],{actionType:`check`,value:r}),c&&C(c,[r]),a.value=[r],i(),s()):(u&&C(u,[],{actionType:`uncheck`,value:r}),l&&C(l,[],{actionType:`uncheck`,value:r}),c&&C(c,[]),a.value=[],i(),s())}return T(k,{checkedCountRef:s,maxRef:y(e,`max`),minRef:y(e,`min`),valueSetRef:c,disabledRef:i,mergedSizeRef:r,toggleCheckbox:l}),{mergedClsPrefix:t}},render(){return t(`div`,{class:`${this.mergedClsPrefix}-checkbox-group`,role:`group`},this.$slots)}}),j=()=>t(`svg`,{viewBox:`0 0 64 64`,class:`check-icon`},t(`path`,{d:`M50.42,16.76L22.34,39.45l-8.1-11.46c-1.12-1.58-3.3-1.96-4.88-0.84c-1.58,1.12-1.95,3.3-0.84,4.88l10.26,14.51  c0.56,0.79,1.42,1.31,2.38,1.45c0.16,0.02,0.32,0.03,0.48,0.03c0.8,0,1.57-0.27,2.2-0.78l30.99-25.03c1.5-1.21,1.74-3.42,0.52-4.92  C54.13,15.78,51.93,15.55,50.42,16.76z`})),M=()=>t(`svg`,{viewBox:`0 0 100 100`,class:`line-icon`},t(`path`,{d:`M80.2,55.5H21.4c-2.8,0-5.1-2.5-5.1-5.5l0,0c0-3,2.3-5.5,5.1-5.5h58.7c2.8,0,5.1,2.5,5.1,5.5l0,0C85.2,53.1,82.9,55.5,80.2,55.5z`})),N=r([u(`checkbox`,`
 font-size: var(--n-font-size);
 outline: none;
 cursor: pointer;
 display: inline-flex;
 flex-wrap: nowrap;
 align-items: flex-start;
 word-break: break-word;
 line-height: var(--n-size);
 --n-merged-color-table: var(--n-color-table);
 `,[l(`show-label`,`line-height: var(--n-label-line-height);`),r(`&:hover`,[u(`checkbox-box`,[d(`border`,`border: var(--n-border-checked);`)])]),r(`&:focus:not(:active)`,[u(`checkbox-box`,[d(`border`,`
 border: var(--n-border-focus);
 box-shadow: var(--n-box-shadow-focus);
 `)])]),l(`inside-table`,[u(`checkbox-box`,`
 background-color: var(--n-merged-color-table);
 `)]),l(`checked`,[u(`checkbox-box`,`
 background-color: var(--n-color-checked);
 `,[u(`checkbox-icon`,[r(`.check-icon`,`
 opacity: 1;
 transform: scale(1);
 `)])])]),l(`indeterminate`,[u(`checkbox-box`,[u(`checkbox-icon`,[r(`.check-icon`,`
 opacity: 0;
 transform: scale(.5);
 `),r(`.line-icon`,`
 opacity: 1;
 transform: scale(1);
 `)])])]),l(`checked, indeterminate`,[r(`&:focus:not(:active)`,[u(`checkbox-box`,[d(`border`,`
 border: var(--n-border-checked);
 box-shadow: var(--n-box-shadow-focus);
 `)])]),u(`checkbox-box`,`
 background-color: var(--n-color-checked);
 border-left: 0;
 border-top: 0;
 `,[d(`border`,{border:`var(--n-border-checked)`})])]),l(`disabled`,{cursor:`not-allowed`},[l(`checked`,[u(`checkbox-box`,`
 background-color: var(--n-color-disabled-checked);
 `,[d(`border`,{border:`var(--n-border-disabled-checked)`}),u(`checkbox-icon`,[r(`.check-icon, .line-icon`,{fill:`var(--n-check-mark-color-disabled-checked)`})])])]),u(`checkbox-box`,`
 background-color: var(--n-color-disabled);
 `,[d(`border`,`
 border: var(--n-border-disabled);
 `),u(`checkbox-icon`,[r(`.check-icon, .line-icon`,`
 fill: var(--n-check-mark-color-disabled);
 `)])]),d(`label`,`
 color: var(--n-text-color-disabled);
 `)]),u(`checkbox-box-wrapper`,`
 position: relative;
 width: var(--n-size);
 flex-shrink: 0;
 flex-grow: 0;
 user-select: none;
 -webkit-user-select: none;
 `),u(`checkbox-box`,`
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
 `,[d(`border`,`
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
 `),u(`checkbox-icon`,`
 display: flex;
 align-items: center;
 justify-content: center;
 position: absolute;
 left: 1px;
 right: 1px;
 top: 1px;
 bottom: 1px;
 `,[r(`.check-icon, .line-icon`,`
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
 `),e({left:`1px`,top:`1px`})])]),d(`label`,`
 color: var(--n-text-color);
 transition: color .3s var(--n-bezier);
 user-select: none;
 -webkit-user-select: none;
 padding: var(--n-label-padding);
 font-weight: var(--n-label-font-weight);
 `,[r(`&:empty`,{display:`none`})])]),x(u(`checkbox`,`
 --n-merged-color-table: var(--n-color-table-modal);
 `)),_(u(`checkbox`,`
 --n-merged-color-table: var(--n-color-table-popover);
 `))]),P=S({name:`Checkbox`,props:Object.assign(Object.assign({},g.props),{size:String,checked:{type:[Boolean,String,Number],default:void 0},defaultChecked:{type:[Boolean,String,Number],default:!1},value:[String,Number],disabled:{type:Boolean,default:void 0},indeterminate:Boolean,label:String,focusable:{type:Boolean,default:!0},checkedValue:{type:[Boolean,String,Number],default:!0},uncheckedValue:{type:[Boolean,String,Number],default:!1},"onUpdate:checked":[Function,Array],onUpdateChecked:[Function,Array],privateInsideTable:Boolean,onChange:[Function,Array]}),setup(e){let t=s(k,null),r=v(null),{mergedClsPrefixRef:i,inlineThemeDisabled:a,mergedRtlRef:o,mergedComponentPropsRef:l}=b(e),u=v(e.defaultChecked),d=D(y(e,`checked`),u),_=n(()=>{if(t){let n=t.valueSetRef.value;return n&&e.value!==void 0?n.has(e.value):!1}else return d.value===e.checkedValue}),x=f(e,{mergedSize(n){let{size:r}=e;if(r!==void 0)return r;if(t){let{value:e}=t.mergedSizeRef;if(e!==void 0)return e}if(n){let{mergedSize:e}=n;if(e!==void 0)return e.value}return l?.value?.Checkbox?.size||`medium`},mergedDisabled(n){let{disabled:r}=e;if(r!==void 0)return r;if(t){if(t.disabledRef.value)return!0;let{maxRef:{value:e},checkedCountRef:n}=t;if(e!==void 0&&n.value>=e&&!_.value)return!0;let{minRef:{value:r}}=t;if(r!==void 0&&n.value<=r&&_.value)return!0}return n?n.disabled.value:!1}}),{mergedDisabledRef:S,mergedSizeRef:w}=x,T=g(`Checkbox`,`-checkbox`,N,O,e,i);function A(n){if(t&&e.value!==void 0)t.toggleCheckbox(!_.value,e.value);else{let{onChange:t,"onUpdate:checked":r,onUpdateChecked:i}=e,{nTriggerFormInput:a,nTriggerFormChange:o}=x,s=_.value?e.uncheckedValue:e.checkedValue;r&&C(r,s,n),i&&C(i,s,n),t&&C(t,s,n),a(),o(),u.value=s}}function j(e){S.value||A(e)}function M(e){if(!S.value)switch(e.key){case` `:case`Enter`:A(e)}}function P(e){switch(e.key){case` `:e.preventDefault()}}let F={focus:()=>{var e;(e=r.value)==null||e.focus()},blur:()=>{var e;(e=r.value)==null||e.blur()}},I=c(`Checkbox`,o,i),L=h(()=>{let{value:e}=w,{common:{cubicBezierEaseInOut:t},self:{borderRadius:n,color:r,colorChecked:i,colorDisabled:a,colorTableHeader:o,colorTableHeaderModal:s,colorTableHeaderPopover:c,checkMarkColor:l,checkMarkColorDisabled:u,border:d,borderFocus:f,borderDisabled:m,borderChecked:h,boxShadowFocus:g,textColor:_,textColorDisabled:v,checkMarkColorDisabledChecked:y,colorDisabledChecked:b,borderDisabledChecked:x,labelPadding:S,labelLineHeight:C,labelFontWeight:E,[p(`fontSize`,e)]:D,[p(`size`,e)]:O}}=T.value;return{"--n-label-line-height":C,"--n-label-font-weight":E,"--n-size":O,"--n-bezier":t,"--n-border-radius":n,"--n-border":d,"--n-border-checked":h,"--n-border-focus":f,"--n-border-disabled":m,"--n-border-disabled-checked":x,"--n-box-shadow-focus":g,"--n-color":r,"--n-color-checked":i,"--n-color-table":o,"--n-color-table-modal":s,"--n-color-table-popover":c,"--n-color-disabled":a,"--n-color-disabled-checked":b,"--n-text-color":_,"--n-text-color-disabled":v,"--n-check-mark-color":l,"--n-check-mark-color-disabled":u,"--n-check-mark-color-disabled-checked":y,"--n-font-size":D,"--n-label-padding":S}}),R=a?m(`checkbox`,h(()=>w.value[0]),L,e):void 0;return Object.assign(x,F,{rtlEnabled:I,selfRef:r,mergedClsPrefix:i,mergedDisabled:S,renderedChecked:_,mergedTheme:T,labelId:E(),handleClick:j,handleKeyUp:M,handleKeyDown:P,cssVars:a?void 0:L,themeClass:R?.themeClass,onRender:R?.onRender})},render(){var e;let{$slots:n,renderedChecked:r,mergedDisabled:a,indeterminate:s,privateInsideTable:c,cssVars:l,labelId:u,label:d,mergedClsPrefix:f,focusable:p,handleKeyUp:m,handleKeyDown:h,handleClick:g}=this;(e=this.onRender)==null||e.call(this);let _=w(n.default,e=>d||e?t(`span`,{class:`${f}-checkbox__label`,id:u},d||e):null);return t(`div`,{ref:`selfRef`,class:[`${f}-checkbox`,this.themeClass,this.rtlEnabled&&`${f}-checkbox--rtl`,r&&`${f}-checkbox--checked`,a&&`${f}-checkbox--disabled`,s&&`${f}-checkbox--indeterminate`,c&&`${f}-checkbox--inside-table`,_&&`${f}-checkbox--show-label`],tabindex:a||!p?void 0:0,role:`checkbox`,"aria-checked":s?`mixed`:r,"aria-labelledby":u,style:l,onKeyup:m,onKeydown:h,onClick:g,onMousedown:()=>{i(`selectstart`,window,e=>{e.preventDefault()},{once:!0})}},t(`div`,{class:`${f}-checkbox-box-wrapper`},`\xA0`,t(`div`,{class:`${f}-checkbox-box`},t(o,null,{default:()=>this.indeterminate?t(`div`,{key:`indeterminate`,class:`${f}-checkbox-icon`},M()):t(`div`,{key:`check`,class:`${f}-checkbox-icon`},j())}),t(`div`,{class:`${f}-checkbox-box__border`}))),_)}});export{A as n,P as t};