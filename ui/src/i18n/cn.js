import deepmerge from 'deepmerge'
import chineseMessages from 'ra-language-chinese'

export default deepmerge(chineseMessages, {
  languageName: '简体中文',
  resources: {
    song: {
      name: '歌曲 |||| 曲库',
      fields: {
        title: '标题',
        artist: '歌手',
        album: '专辑',
        path: '路径',
        genre: '类型',
        compilation: '收录',
        albumArtist: '专辑歌手',
        duration: '时长',
        year: '年份',
        playCount: '播放次数',
        trackNumber: '音轨 #',
        size: '大小',
        updatedAt: '上次更新',
      },
      bulk: {
        addToQueue: '稍后播放',
      },
    },
    album: {
      name: '专辑 |||| 专辑',
      fields: {
        name: '名称',
        albumArtist: '专辑歌手',
        artist: '歌手',
        duration: '时长',
        songCount: '曲目数',
        playCount: '播放次数',
        compilation: '合辑',
        year: '年份',
      },
      actions: {
        playAll: '播放',
        playNext: '播放下一首',
        addToQueue: '稍后播放',
        shuffle: '刷新',
      },
    },
    artist: {
      name: '歌手 |||| 歌手',
      fields: {
        name: '名称',
        albumCount: '歌手数',
      },
    },
    user: {
      name: '用户 |||| 用户',
      fields: {
        userName: '用户名',
        isAdmin: '管理员',
        lastLoginAt: '最后一次访问',
        updatedAt: '上次修改',
        name: '名称',
      },
    },
    player: {
      name: '用户 |||| 用户',
      fields: {
        name: '名称',
        transcodingId: '转码',
        maxBitRate: '最大比特率',
        client: '应用程序',
        userName: '用户',
        lastSeen: '最后一次访问',
      },
    },
    transcoding: {
      name: '转码 |||| 转码',
      fields: {
        name: '名称',
        targetFormat: '格式',
        defaultBitRate: '默认比特率',
        command: '命令',
      },
    },
  },
  ra: {
    auth: {
      welcome1: '感谢您安装Navidrome!',
      welcome2: '为了开始使用,请创建一个管理员账户',
      confirmPassword: '确认密码',
      buttonCreateAdmin: '创建管理员',
    },
    validation: {
      invalidChars: '请只使用字母和数字',
      passwordDoesNotMatch: '密码不匹配',
    },
  },
  menu: {
    library: '曲库',
    settings: '设置',
    version: '版本 %{version}',
    theme: '主题',
    personal: {
      name: '个性化',
      options: {
        theme: '主题',
        language: '语言',
      },
    },
  },
  player: {
    playListsText: '播放队列',
    openText: '打开',
    closeText: '关闭',
    notContentText: '无音乐',
    clickToPlayText: '点击播放',
    clickToPauseText: '点击暂停',
    nextTrackText: '下一首',
    previousTrackText: '上一首',
    reloadText: 'Reload',
    volumeText: '音量',
    toggleLyricText: '切换歌词',
    toggleMiniModeText: '最小化',
    destroyText: '损坏',
    downloadText: '下载',
    removeAudioListsText: '清空播放列表',
    controllerTitle: '',
    clickToDeleteText: `点击删除 %{name}`,
    emptyLyricText: '无歌词',
    playModeText: {
      order: '顺序播放',
      orderLoop: '列表循环',
      singleLoop: '单曲循环',
      shufflePlay: '随机播放',
    },
  },
})
