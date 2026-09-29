# transformer.py 代码讲解

这份代码用 PyTorch 实现了 Transformer 的几个核心组件，按从小到大的顺序是：

| 模块 | 作用 | 在 Transformer 里的位置 |
|---|---|---|
| `PositionalEncoding` | 给词向量加上位置信息 | 输入嵌入之后 |
| `MultiHeadAttention` | 多头注意力 | 编码器、解码器的每一层 |
| `PositionWiseFeedForward` | 逐位置前馈网络 | 编码器、解码器的每一层 |
| `EncoderLayer` | 一层编码器 = 自注意力 + 前馈 | 编码器堆叠 N 层 |
| `DecoderLayer` | 一层解码器 = 掩码自注意力 + 交叉注意力 + 前馈 | 解码器堆叠 N 层 |

下文中常用的符号：

- `batch_size`：一批里有几个句子
- `seq_len`：句子长度（token 个数）
- `d_model`：每个 token 向量的维度（论文里是 512）
- `num_heads`：注意力头数（论文里是 8）
- `d_k = d_model / num_heads`：每个头分到的维度（论文里是 64）
- `d_ff`：前馈网络中间层维度（论文里是 2048）

---

## 1. 导入

```python
import torch
import torch.nn as nn
import math
```

- `torch`：张量计算、自动求导
- `torch.nn`：神经网络层（`Linear`、`LayerNorm`、`Dropout` 等）和基类 `nn.Module`
- `math`：用到 `math.log`、`math.sqrt`

所有模块都继承 `nn.Module`，并在 `__init__` 开头调用 `super().__init__()`，让 PyTorch 能登记子层和参数（这样 `model.parameters()`、`.to(device)` 才能找到它们）。

---

## 2. PositionalEncoding：位置编码

### 为什么需要

注意力机制本身不关心顺序：把句子里的词打乱，每个词算出来的注意力结果不变。所以要额外给每个位置一个"位置向量"，加到词向量上，让模型知道谁在前、谁在后。

### 公式

论文使用固定的正弦/余弦函数：

```
PE(pos, 2i)   = sin(pos / 10000^(2i / d_model))
PE(pos, 2i+1) = cos(pos / 10000^(2i / d_model))
```

- `pos`：第几个位置（0, 1, 2, …）
- `i`：第几对维度；偶数维用 sin，奇数维用 cos
- 低维度频率高、变化快，高维度频率低、变化慢，组合起来每个位置都有唯一的编码

### 逐行解读

```python
position = torch.arange(max_len).unsqueeze(1)
```
生成 `[0, 1, …, max_len-1]`，`unsqueeze(1)` 变成形状 `(max_len, 1)` 的列向量，方便后面广播。

```python
div_term = torch.exp(torch.arange(0, d_model, 2) * (-math.log(10000.0) / d_model))
```
计算 `1 / 10000^(2i/d_model)`，形状 `(d_model/2,)`。
这里用 `exp(-2i · ln10000 / d_model)` 代替直接求幂，是一个等价写法，数值上更稳定。

```python
pe = torch.zeros(max_len, d_model)
pe[:, 0::2] = torch.sin(position * div_term)
pe[:, 1::2] = torch.cos(position * div_term)
```
`position * div_term` 广播成 `(max_len, d_model/2)`，然后：
- `0::2` 表示从第 0 列开始每隔一列取（偶数列），填 sin
- `1::2` 表示从第 1 列开始每隔一列取（奇数列），填 cos

```python
self.register_buffer('pe', pe.unsqueeze(0))
```
- `unsqueeze(0)` 加一个 batch 维，形状变为 `(1, max_len, d_model)`
- `register_buffer`：把 `pe` 存进模块，**不作为可训练参数**（优化器不会更新它），但会随 `.to(device)` 移动，也会保存到 `state_dict` 里

### forward

```python
x = x + self.pe[:, :x.size(1)]
return self.dropout(x)
```
- `x` 形状 `(batch_size, seq_len, d_model)`
- 只取前 `seq_len` 个位置的编码，形状 `(1, seq_len, d_model)`，广播加到每个样本上
- 最后做 dropout，防止过拟合

---

## 3. MultiHeadAttention：多头注意力

这是 Transformer 最核心的部分。

### 直观理解

每个 token 会生成三个向量：
- **Q（Query，查询）**：我要找什么
- **K（Key，键）**：我能提供什么
- **V（Value，值）**：我的实际内容

用 Q 去和所有 K 做点积，得到"相关程度"，softmax 成权重，再用权重对 V 加权求和，得到这个 token 的新表示。

"多头"是指把 `d_model` 切成 `num_heads` 份，每个头在自己的子空间里独立做注意力，可以同时关注不同类型的关系（比如一个头看语法、一个头看指代），最后再拼回来。

### `__init__`

```python
assert d_model % num_heads == 0
self.d_k = d_model // num_heads

self.W_q = nn.Linear(d_model, d_model)
self.W_k = nn.Linear(d_model, d_model)
self.W_v = nn.Linear(d_model, d_model)
self.W_o = nn.Linear(d_model, d_model)
```
- `d_model` 必须能被 `num_heads` 整除，否则无法平均切分
- `W_q / W_k / W_v`：把输入分别投影成 Q、K、V
- `W_o`：多个头拼接后，再做一次线性变换输出
- 这里用一个 `d_model → d_model` 的大矩阵，等价于每个头各有一个 `d_model → d_k` 的小矩阵，一次算完效率更高

### scaled_dot_product_attention：缩放点积注意力

公式：

```
Attention(Q, K, V) = softmax(Q · Kᵀ / √d_k) · V
```

```python
attn_scores = torch.matmul(Q, K.transpose(-2, -1)) / math.sqrt(self.d_k)
```
- `K.transpose(-2, -1)`：交换最后两个维度，即 Kᵀ
- 结果形状 `(batch, heads, seq_len_q, seq_len_k)`，第 `[i][j]` 个值表示第 i 个词对第 j 个词的关注分数
- **除以 √d_k 的原因**：d_k 越大，点积的数值越大，softmax 会变得非常尖锐（接近 one-hot），梯度几乎为 0，训练不动。缩放后分数回到合理范围

```python
if mask is not None:
    attn_scores = attn_scores.masked_fill(mask == 0, -1e9)
```
把不允许关注的位置设成一个很大的负数，softmax 之后这些位置的权重约等于 0。常见两种掩码：
- **padding mask**：忽略补齐用的 `<pad>` token
- **look-ahead mask（因果掩码）**：解码器里不让当前位置看到后面的词，防止"偷看答案"

```python
attn_probs = torch.softmax(attn_scores, dim=-1)
output = torch.matmul(attn_probs, V)
```
- 在最后一维做 softmax，每一行加起来等于 1，变成权重
- 权重乘 V，得到加权后的结果，形状 `(batch, heads, seq_len_q, d_k)`

### split_heads：拆分多头

```python
return x.view(batch_size, seq_length, self.num_heads, self.d_k).transpose(1, 2)
```
形状变化：

```
(batch, seq_len, d_model)
  → view      → (batch, seq_len, num_heads, d_k)   把最后一维切成 num_heads 段
  → transpose → (batch, num_heads, seq_len, d_k)   把"头"提到前面
```
把 heads 放到第 2 维之后，后面的 `matmul` 会对每个头独立计算（前两维当作批次维）。

### combine_heads：合并多头

```python
return x.transpose(1, 2).contiguous().view(batch_size, seq_length, self.d_model)
```
split_heads 的逆操作：

```
(batch, num_heads, seq_len, d_k)
  → transpose → (batch, seq_len, num_heads, d_k)
  → view      → (batch, seq_len, d_model)
```
- `transpose` 之后张量在内存中不再连续，`view` 要求内存连续，所以中间要调用 `.contiguous()` 复制一份连续内存

### forward：整体流程

```python
Q = self.split_heads(self.W_q(Q))
K = self.split_heads(self.W_k(K))
V = self.split_heads(self.W_v(V))
attn_output = self.scaled_dot_product_attention(Q, K, V, mask)
output = self.W_o(self.combine_heads(attn_output))
```

```
输入 Q, K, V (batch, seq_len, d_model)
   │  线性投影 W_q / W_k / W_v
   ▼
split_heads → (batch, heads, seq_len, d_k)
   │
   ▼
缩放点积注意力 → (batch, heads, seq_len, d_k)
   │
   ▼
combine_heads → (batch, seq_len, d_model)
   │  线性投影 W_o
   ▼
输出 (batch, seq_len, d_model)
```

输入和输出形状相同，所以可以在后面加残差连接。

forward 把 Q、K、V 作为三个参数分开传入，这样同一个模块可以用于两种场景：
- **自注意力**：`self_attn(x, x, x, mask)`，Q、K、V 都来自同一个序列
- **交叉注意力**：`cross_attn(x, enc_out, enc_out, mask)`，Q 来自解码器，K、V 来自编码器

---

## 4. PositionWiseFeedForward：逐位置前馈网络

```python
self.linear1 = nn.Linear(d_model, d_ff)
self.relu = nn.ReLU()
self.dropout = nn.Dropout(dropout)
self.linear2 = nn.Linear(d_ff, d_model)
```

公式：

```
FFN(x) = Linear2(Dropout(ReLU(Linear1(x))))
```

- 先把维度从 `d_model` 放大到 `d_ff`（通常是 4 倍），经过 ReLU 激活，再压回 `d_model`
- **"逐位置"（Position-wise）**：`nn.Linear` 只作用在最后一维，所以每个 token 各自独立地经过同一个网络，token 之间不交换信息
- 分工：注意力层负责 token **之间**的信息交互，前馈层负责对每个 token **自身**做非线性变换

形状：`(batch, seq_len, d_model) → (batch, seq_len, d_ff) → (batch, seq_len, d_model)`

---

## 5. EncoderLayer：编码器层

```python
def forward(self, x, mask):
    attn_output = self.self_attn(x, x, x, mask)
    x = self.norm1(x + self.dropout(attn_output))

    ff_output = self.feed_forward(x)
    x = self.norm2(x + self.dropout(ff_output))
    return x
```

一层编码器包含两个子层，每个子层都套了同一个结构：

```
x = LayerNorm(x + Dropout(子层(x)))
```

- **残差连接 `x + 子层(x)`**：让梯度可以直接沿着 `x` 往回传，缓解深层网络难训练的问题；子层只需要学习"在原输入基础上改多少"
- **LayerNorm**：对每个 token 的 `d_model` 维做归一化（均值 0、方差 1，再加可学习的缩放和偏移），让数值分布稳定
- 这种"先相加，再归一化"的写法叫 **Post-LN**，与原论文一致。现在很多大模型改用 Pre-LN：`x + 子层(LayerNorm(x))`，训练更稳定

流程：

```
x ──► 多头自注意力 ──► Dropout ──► (+x) ──► LayerNorm
  ──► 前馈网络     ──► Dropout ──► (+x) ──► LayerNorm ──► 输出
```

---

## 6. DecoderLayer：解码器层

```python
def forward(self, x, encoder_output, src_mask, tgt_mask):
    attn_output = self.self_attn(x, x, x, tgt_mask)
    x = self.norm1(x + self.dropout(attn_output))

    cross_attn_output = self.cross_attn(x, encoder_output, encoder_output, src_mask)
    x = self.norm2(x + self.dropout(cross_attn_output))

    ff_output = self.feed_forward(x)
    x = self.norm3(x + self.dropout(ff_output))
    return x
```

比编码器多一个子层，共三个：

1. **掩码多头自注意力**（`tgt_mask`）：解码器对已生成的目标序列做自注意力。`tgt_mask` 通常是 padding mask 加上因果掩码（下三角矩阵），保证第 i 个位置只能看到 0 ~ i 的词
2. **交叉注意力**（`src_mask`）：Q 来自解码器当前状态，K、V 来自 `encoder_output`，相当于"拿着目标端的问题去源句子里找信息"。`src_mask` 用来屏蔽源句子里的 padding
3. **前馈网络**：和编码器一样

每个子层同样是 `LayerNorm(x + Dropout(子层(x)))`，所以有 `norm1 / norm2 / norm3` 三个归一化层。

---

## 7. 已修复的问题

最初的代码有两个问题，导致编码器层和解码器层无法使用，现已修复：

### 问题 1：`MultiHeadAttention` 里有一个重复的 `forward` 覆盖了正确实现

类的末尾残留了一段占位代码：

```python
    def forward(self, query, key, value, mask):
        pass
```

Python 类里同名方法后定义的会覆盖先定义的，所以生效的是这个空的 `forward`，调用 `MultiHeadAttention` 永远返回 `None`。**修复：删除这段占位代码。**

### 问题 2：`EncoderLayer` / `DecoderLayer` 创建子模块时没有传参数

```python
self.self_attn = MultiHeadAttention()          # 缺少 d_model, num_heads
self.feed_forward = PositionWiseFeedForward()  # 缺少 d_model, d_ff
```

运行时会报错：

```
TypeError: MultiHeadAttention.__init__() missing 2 required positional arguments: 'd_model' and 'num_heads'
```

**修复：** 把构造参数传进去：

```python
# EncoderLayer
self.self_attn = MultiHeadAttention(d_model, num_heads)
self.feed_forward = PositionWiseFeedForward(d_model, d_ff, dropout)

# DecoderLayer
self.self_attn = MultiHeadAttention(d_model, num_heads)
self.cross_attn = MultiHeadAttention(d_model, num_heads)
self.feed_forward = PositionWiseFeedForward(d_model, d_ff, dropout)
```

### 验证

```python
x = torch.randn(2, 5, 16)                       # batch=2, seq_len=5, d_model=16
enc = EncoderLayer(16, 4, 64, 0.1)(x, None)     # → (2, 5, 16)
mask = torch.tril(torch.ones(5, 5)).bool()      # 因果掩码（下三角）
dec = DecoderLayer(16, 4, 64, 0.1)(x, enc, None, mask)  # → (2, 5, 16)
```

输出形状与输入一致，说明各层可以正常串联。

### 还缺少的部分

这份代码只实现了"一层"，要组成完整的 Transformer，还需要：

- 词嵌入层 `nn.Embedding`（通常乘以 `√d_model`）
- 把 N 个 `EncoderLayer` / `DecoderLayer` 堆叠起来的 `Encoder` / `Decoder`
- 生成 `src_mask` 和 `tgt_mask`（padding mask、因果掩码）的函数
- 最后的输出层：`nn.Linear(d_model, vocab_size)`，再经过 softmax 得到每个词的概率

---

## 8. 整体结构图

```
          源序列                               目标序列（右移一位）
            │                                        │
       词嵌入 + 位置编码                          词嵌入 + 位置编码
            │                                        │
   ┌────────▼────────┐                     ┌─────────▼─────────┐
   │   EncoderLayer  │                     │  掩码多头自注意力   │
   │  ┌───────────┐  │                     │   + 残差 & LN      │
   │  │ 多头自注意力 │  │                     ├───────────────────┤
   │  │ + 残差 & LN │  │   encoder_output    │    交叉注意力       │
   │  ├───────────┤  │ ──────────────────► │   + 残差 & LN      │
   │  │  前馈网络   │  │     (作为 K, V)       ├───────────────────┤
   │  │ + 残差 & LN │  │                     │     前馈网络        │
   │  └───────────┘  │                     │   + 残差 & LN      │
   └─────────────────┘                     └─────────┬─────────┘
        × N 层                                  × N 层│
                                                     ▼
                                           Linear + Softmax
                                                     │
                                                 输出概率
```
