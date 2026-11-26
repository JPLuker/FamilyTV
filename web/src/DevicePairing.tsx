import {useEffect,useState} from 'react'
import {errorMessage,request} from './api'

export default function DevicePairing({challenge,onPaired,profileRevision}:{
  challenge?:string;onPaired:()=>void;profileRevision:number
}){
  const path='devices'
  const [screens,setScreens]=useState<{id:string;profile:string;allow_start:boolean}[]>([])
  const [profiles,setProfiles]=useState<{id:string;name:string}[]>([])
  useEffect(()=>{
    let live=true
    request<{profiles:{id:string;name:string}[]}>('profiles').then(r=>{if(live)setProfiles(r.profiles||[])}).catch(e=>{if(live)setNotice(errorMessage(e))})
    return ()=>{live=false}
  },[profileRevision])
  const [allowStart,setAllowStart]=useState(false)
  const [code,setCode]=useState('')
  const [notice,setNotice]=useState('')
  const [busy,setBusy]=useState(false)
  useEffect(()=>{
    let mounted=true
    async function refresh(){
      try{const value=await request<{devices:typeof screens}>(path);if(mounted)setScreens(value.devices)}
      catch(e){if(mounted)setNotice(errorMessage(e))}
    }
    void refresh()
    const timer=setInterval(refresh,5000)
    return()=>{mounted=false;clearInterval(timer)}
  },[path])
  async function change(method:'POST'|'PUT'|'DELETE'){
    if(method==='DELETE'&&!window.confirm('Disconnect every device from this household?'))return
    setBusy(true);setNotice('')
    try{
      const result=await request<{code?:string}>(path,method,method==='DELETE'?undefined:{challenge:method==='PUT'?challenge:undefined,allow_start:allowStart})
      setCode(result.code||'')
      if(method==='DELETE'){setScreens([]);setNotice('Devices disconnected.')}
      if(method==='PUT'){onPaired();setNotice('TV authorized. Choose Open paired viewer on the TV.')}
    }catch(e){setNotice(errorMessage(e))}finally{setBusy(false)}
  }
  return <section aria-label="Household devices"><h2>Household devices</h2>
    <p>Link phones, tablets and TVs to your account. Assign a viewing profile when the screen is ready to use.</p>
    <label className="toggle"><input type="checkbox" checked={allowStart} onChange={e=>setAllowStart(e.target.checked)}/> Allow the new device to start a one-video session with the parent PIN</label>
    <p>Set your PIN under Parent sign-in settings.</p>
    {challenge&&<button disabled={busy} onClick={()=>change('PUT')}>Pair TV {challenge.slice(0,8)}</button>}
    <button disabled={busy} onClick={()=>change('POST')}>{code?'Replace pairing code':'Get pairing code'}</button>
    {code&&<><p>Device pairing code <strong>{code}</strong> · valid for 10 minutes</p>
      <label>One-time viewer link<input readOnly value={location.origin+'/watch#'+code} onFocus={e=>e.target.select()}/></label>
      <p>Use this link in the child’s browser. Opening it in the parent browser signs you out.</p></>}
    {screens.map(screen=><div key={screen.id}><label>Viewing profile for screen {screen.id.slice(0,6)}
      <select disabled={busy} value={screen.profile} onChange={async e=>{
        const profile=e.target.value
        setBusy(true);setNotice('')
        try{await request(path,'PATCH',{device:screen.id,profile});setScreens(old=>old.map(d=>d.id===screen.id?{...d,profile}:d))}
        catch(err){setNotice(errorMessage(err))}finally{setBusy(false)}
      }}><option value="">Waiting for assignment</option>{profiles.map(p=><option key={p.id} value={p.id}>{p.name}</option>)}</select>
    </label><label className="toggle"><input type="checkbox" disabled={busy} checked={screen.allow_start} onChange={async e=>{
      const allow_start=e.target.checked
      setBusy(true)
      try{await request(path,'PATCH',{device:screen.id,profile:screen.profile,allow_start});setScreens(old=>old.map(d=>d.id===screen.id?{...d,allow_start}:d))}
      catch(err){setNotice(errorMessage(err))}finally{setBusy(false)}
    }}/> Allow PIN-protected starts on screen {screen.id.slice(0,6)}</label></div>)}
    {(screens.length>0||code)&&<button disabled={busy} onClick={()=>change('DELETE')}>Disconnect all devices</button>}
    {notice&&<p role="status">{notice}</p>}

  </section>
}
