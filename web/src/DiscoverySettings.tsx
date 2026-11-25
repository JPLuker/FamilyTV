export type Discovery={unique_creators?:boolean;creator_cooldown?:boolean;mix_subjects?:boolean;pace_series?:boolean;reading?:string;preset?:string;choices?:number;preferred_minutes?:number;search?:string;interests?:string;spoken_labels?:boolean;warn_minutes?:number}
export const discoveryDefaults:Discovery={preset:'',choices:12,preferred_minutes:0,search:'on',interests:'',spoken_labels:false,warn_minutes:0}
const presets:Record<string,[number,number,string]>={'2-4':[4,5,'off'],'5-7':[6,10,'on'],'8-11':[8,15,'on'],'12-14':[12,20,'on'],'15-17':[12,0,'on']}
export default function DiscoverySettings({value,change}:{value:Discovery;change:(value:Discovery)=>void}){
  return <details><summary>Discovery preferences</summary>
    <p>Age presets suggest a number of choices and video length. You can change either at any age. Choose reading detail, subjects and sensory settings for this child. Preferred length changes the order of choices. Session deadlines still apply.</p>
    <label>Starting age preset<select value={value.preset||''} onChange={e=>{
      const preset=e.target.value,defaults=presets[preset]
      change(defaults?{...value,preset,choices:defaults[0],preferred_minutes:defaults[1],search:defaults[2]}:{...value,preset})
    }}><option value="">Custom</option>{Object.keys(presets).map(key=><option key={key} value={key}>{key} years</option>)}</select></label>
    {([['unique_creators','One choice per creator on the board'],['creator_cooldown','Suggest other creators first after playback'],['mix_subjects','Alternate subjects in suggestions'],['pace_series','Spread episodes from the same series apart']] as const).map(([key,label])=><label className="toggle" key={key}><input type="checkbox" checked={Boolean(value[key])} onChange={e=>change({...value,[key]:e.target.checked})}/>{label}</label>)}
    <label>Reading detail<select value={value.reading||'brief'} onChange={e=>change({...value,reading:e.target.value})}><option value="brief">Short labels</option><option value="detail">Include the subject</option></select></label>
    <label>Choices on the board<select value={value.choices||12} onChange={e=>change({...value,choices:Number(e.target.value)})}>{[4,6,8,12].map(n=><option key={n}>{n}</option>)}</select></label>
    <label>Preferred video length in minutes (0 for any)<input type="number" min="0" max="240" value={value.preferred_minutes||0} onChange={e=>change({...value,preferred_minutes:Number(e.target.value)})}/></label>
    <label>Favorite subjects, separated by commas<input maxLength={250} value={value.interests||''} onChange={e=>change({...value,interests:e.target.value})}/></label>
    <label>Transition warning in minutes (0 for none)<input type="number" min="0" max="15" value={value.warn_minutes||0} onChange={e=>change({...value,warn_minutes:Number(e.target.value)})}/></label>
    <label className="toggle"><input type="checkbox" checked={Boolean(value.spoken_labels)} onChange={e=>change({...value,spoken_labels:e.target.checked})}/>Speak focused titles using an available device voice</label>
  </details>
}
